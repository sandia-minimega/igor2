// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func dbCreateHosts(nodes []Host, tx *gorm.DB) error {
	result := tx.Create(&nodes)
	return result.Error
}

// dbReadHostsTx performs dbReadHosts within a new transaction.
func dbReadHostsTx(queryParams map[string]interface{}) (hosts []Host, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		hosts, err = dbReadHosts(queryParams, tx)
		return err
	})

	return hosts, err
}

// dbReadHosts returns a list of hosts that match the given queryParams, possibly returning
// no matches. If no queryParams are provided, all hosts are returned.
func dbReadHosts(queryParams map[string]interface{}, tx *gorm.DB) (hosts []Host, err error) {

	tx = tx.Preload("Cluster").Preload("HostPolicy").Preload("HostPolicy.AccessGroups").
		Preload("Reservations")

	// if no params given, return all
	if len(queryParams) == 0 {
		result := tx.Find(&hosts)
		return hosts, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case bool, string, int, HostState:
			tx = tx.Where(key, val)
		case []int, []HostState:
			if strings.ToLower(key) == "reservations" {
				tx = tx.Joins("JOIN reservations_hosts ON reservations_hosts.host_id = ID AND reservation_id IN ?", val)
			} else {
				tx = tx.Where(key+" IN ?", val)
			}
		case []string:
			tx = tx.Where(key+" IN ?", val)
		default:
			logger.Error().Msgf("dbReadHosts: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}
	result := tx.Find(&hosts)

	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].SequenceID < hosts[j].SequenceID
	})

	return hosts, result.Error
}

// dbEditHosts iterates through a list of hosts applying the same changes to each.
func dbEditHosts(hosts []Host, changes map[string]interface{}, tx *gorm.DB) error {
	if _, hpOK := changes["HostPolicy"]; hpOK {
		hp := changes["HostPolicy"].(HostPolicy)
		for _, h := range hosts {
			if hpUpdateErr := tx.Model(&h).Association("HostPolicy").Append(&hp); hpUpdateErr != nil {
				return hpUpdateErr
			}
		}
		delete(changes, "HostPolicy")
	}
	if len(changes) > 0 {
		result := tx.Model(&hosts).Updates(changes)
		return result.Error
	}
	return nil
}

// dbDeleteHosts removes the list of hosts from the DB
func dbDeleteHosts(targets []Host, tx *gorm.DB) error {
	if len(targets) == 0 {
		return nil
	}
	result := tx.Delete(&targets)
	return result.Error
}

// dbCheckHostAvailable takes a list of hostnames and reports back if any are in a state that don't allow new reservations
// to be made. If status return is 200/OK, it is assumed all the named hosts are available for scheduling.
func dbCheckHostAvailable(hosts []string, tx *gorm.DB) (int, error) {

	var hostsCurrUnavail []string

	// Check if any of the declared hosts are currently not accepting reservations (draining, blocked or error)
	result := tx.Model(&Host{}).
		Where("name IN ? AND state > ?", hosts, HostReserved).
		Pluck("name", &hostsCurrUnavail)
	if result.RowsAffected > 0 {
		return http.StatusConflict, fmt.Errorf("the following hosts are not available at this time: %v", hostsCurrUnavail)
	}

	return http.StatusOK, nil
}

// syncNodes determines if any cluster has already been configured and takes appropriate action
func syncNodes(hostList []Host) {

	logger.Debug().Msg("Syncing nodes")

	if len(hostList) == 0 {
		// Igor hasn't had a cluster configured yet

		noHosts := fmt.Sprintf("no hosts defined -- if this is a new igor instance an admin must supply a host config file via igor client")
		fmt.Println(noHosts)
		logger.Warn().Msg(noHosts)
		clusterConfigLocHome := filepath.Join(igor.IgorHome, "conf", IgorClusterConfDefault)

		if _, errArg := os.Stat(IgorClusterConfPathDefault); errArg == nil {
			igor.ClusterConfPath = IgorClusterConfPathDefault
			logger.Info().Msgf("cluster configuration file found at %s, use client to configure", igor.ClusterConfPath)
		} else if _, errEtc := os.Stat(clusterConfigLocHome); errEtc == nil {
			igor.ClusterConfPath = clusterConfigLocHome
			logger.Info().Msgf("cluster configuration file found at %s, use client to configure", igor.ClusterConfPath)
		} else {

		}
	} else {
		// Hosts were found in the DB, send cluster info to log
		for _, r := range igor.ClusterRefs {
			logger.Info().Msgf("cluster %s : %s[%d-%d] found", hostList[0].Cluster.Name, r.Prefix, r.Min, r.Max)
		}
		hostReport := fmt.Sprintf("host total = %d", len(hostList))
		logger.Info().Msg(hostReport)
	}

}
