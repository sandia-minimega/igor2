// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func dbCreateCluster(cluster *[]Cluster, tx *gorm.DB) error {
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(cluster)
	return result.Error
}

func dbUpdateClusterDimensions(clusterId int, dimUpdate map[string]interface{}, tx *gorm.DB) error {
	result := tx.Model(&Cluster{}).Where("id = ?", clusterId).Updates(dimUpdate)
	return result.Error
}

func dbReadClustersTx(queryParams map[string]interface{}) (clusters []Cluster, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		clusters, err = dbReadClusters(queryParams, tx)
		return err
	})

	return clusters, err
}

func dbReadClusters(queryParams map[string]interface{}, tx *gorm.DB) (clusters []Cluster, err error) {

	tx = tx.Preload("Hosts.HostPolicy").Preload(clause.Associations)

	if len(queryParams) == 0 {
		result := tx.Find(&clusters)
		return clusters, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string, int:
			tx = tx.Where(key, val)
		case []string, []int:
			tx = tx.Where(key+" IN ?", val)
		default:
			logger.Error().Msgf("dbReadClusters: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}

	result := tx.Find(&clusters)
	return clusters, result.Error
}

func dbUpdateMotdTx(clusterName string, motd string, motdUrgent bool) (err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		result := tx.Model(&Cluster{}).Where("name = ?", clusterName).Updates(map[string]interface{}{"motd": motd, "motd_urgent": motdUrgent})
		return result.Error
	})
	return err
}
