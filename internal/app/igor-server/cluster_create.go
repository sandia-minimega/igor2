// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/hlog"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func doCreateClusters(r *http.Request) (clusters []Cluster, hostnameList []string, status int, err error) {

	clog := hlog.FromRequest(r)
	status = http.StatusInternalServerError // default status, overridden at end if no errors
	var existingHostMsg string
	var dimensionsUpdated bool
	var cConfigs []ClusterConfig

	if err = performDbTx(func(tx *gorm.DB) error {

		var doc []byte
		clusterConfigLocHome := filepath.Join(igor.IgorHome, "conf", IgorClusterConfDefault)

		if _, pathErr := os.Stat(IgorClusterConfPathDefault); pathErr == nil {
			doc, _ = os.ReadFile(IgorClusterConfPathDefault)

		} else if _, pathErr = os.Stat(clusterConfigLocHome); pathErr == nil {
			doc, _ = os.ReadFile(clusterConfigLocHome)

		} else {
			err = fmt.Errorf("could not find cluster config file at %s or %s", IgorClusterConfPathDefault, clusterConfigLocHome)
		}

		if err != nil {
			status = http.StatusNotFound
			return err
		}

		ccMap := make(map[string]ClusterConfig)

		if err = yaml.NewDecoder(bytes.NewReader(doc)).Decode(&ccMap); err != nil {
			clog.Error().Msgf("%v", err)
			return err // uses default err status
		}

		if len(ccMap) > 1 {
			status = http.StatusNotImplemented
			return fmt.Errorf("support for multiple clusters not implemented at this time")
		}

		cluster := Cluster{}
		clusterId := -1

		for cName, cConfig := range ccMap {

			if clusters, err = dbReadClusters(nil, tx); err != nil {
				return err // uses default err status
			} else if len(clusters) > 0 && clusters[0].Name != cName {
				status = http.StatusNotImplemented
				return fmt.Errorf("cluster '%s' already exists, support for multiple clusters not implemented yet", clusters[0].Name)
			} else if len(clusters) > 0 && clusters[0].Name == cName {
				clusterId = clusters[0].ID
				dimUpdate := make(map[string]interface{})
				if cConfig.DisplayWidth != clusters[0].DisplayWidth {
					dimUpdate["DisplayWidth"] = cConfig.DisplayWidth
				}
				if cConfig.DisplayHeight != clusters[0].DisplayHeight {
					dimUpdate["DisplayHeight"] = cConfig.DisplayHeight
				}
				if len(dimUpdate) > 0 {
					if upErr := dbUpdateClusterDimensions(clusterId, dimUpdate, tx); upErr != nil {
						return fmt.Errorf("failed to update cluster dimensions for %s", cName)
					}
					dimensionsUpdated = true
					clog.Info().Msgf(cName+": updated cluster display dimensions to w=%d h=%d", cConfig.DisplayWidth, cConfig.DisplayHeight)
				}
				cConfigs = append(cConfigs, cConfig)
			} else {
				cluster.Name = cName
				cluster.Prefix = cConfig.Prefix
				cluster.DisplayWidth = cConfig.DisplayWidth
				cluster.DisplayHeight = cConfig.DisplayHeight

				cList := []Cluster{cluster}
				err3 := dbCreateCluster(&cList, tx)
				if err3 != nil {
					return err3 // uses default err status
				}
				clusterId = cList[0].ID
				cConfigs = append(cConfigs, cConfig)
			}
		}

		var hostList []Host
		var hostPolicyMap = make(map[string]HostPolicy)

		// at start always add the default policy to the policy map
		if hostPolicyList, err2 := dbReadHostPolicies(map[string]interface{}{"name": DefaultPolicyName}, tx, clog); err2 != nil {
			return err2 // uses default err status
		} else {
			hostPolicyMap[DefaultPolicyName] = hostPolicyList[0]
		}

		for _, v := range ccMap {
			for nmk, nmv := range v.HostMap {
				var hostPolicyName string

				// host's name follows convention <prefix><seq#>
				hname := v.Prefix + strconv.Itoa(nmk)

				hostname := nmv["hostname"]
				// use host's name as hostname if none given
				if hostname == "" {
					hostname = hname
				}

				// mac address is required
				macAddy := nmv["mac"]

				if macAddy == "" {
					status = http.StatusBadRequest
					return fmt.Errorf("required mac address not found for host %s; host configuration aborted", hostname)
				}

				hwAddr, pErr := net.ParseMAC(macAddy)
				if pErr != nil {
					status = http.StatusBadRequest
					return fmt.Errorf("'%s' is not a valid mac address for host %v; host configuration aborted", macAddy, hostname)
				}

				// default is used if no policy is specified in config
				// else look up policy by name in the map
				// else look up policy in db once and add to map for subsequent uses of same policy
				if len(nmv["policy"]) == 0 {
					hostPolicyName = DefaultPolicyName
				} else if _, ok := hostPolicyMap[nmv["policy"]]; ok {
					hostPolicyName = nmv["policy"]
				} else {
					hostPolicyList, rhpErr := dbReadHostPolicies(map[string]interface{}{"name": nmv["policy"]}, tx, clog)
					if rhpErr != nil {
						return rhpErr
					}
					if len(hostPolicyList) == 0 {
						status = http.StatusBadRequest
						return fmt.Errorf("no host policy found with name %s; host configuration aborted", nmv["policy"])
					}
					hostPolicyMap[nmv["policy"]] = hostPolicyList[0]
					hostPolicyName = nmv["policy"]
				}

				hostIP := net.ParseIP(nmv["ip"])
				if hostIP == nil {
					status = http.StatusBadRequest
					return fmt.Errorf("required IP address bad or not found for host %s; host configuration aborted", hostname)
				}
				hostIpBytes := hostIP.String()

				bootMode := nmv["bootMade"]
				if !validBootMode(bootMode) {
					return fmt.Errorf("required bootMode \"%s\" invalid or not found for host %s; host configuration aborted", bootMode, hostname)
				}

				host := &Host{
					Name:         hname,
					HostName:     hostname,
					Eth:          nmv["eth"],
					SequenceID:   nmk,
					Mac:          hwAddr.String(),
					IP:           hostIpBytes,
					BootMode:     bootMode,
					State:        HostBlocked,
					HostPolicyID: hostPolicyMap[hostPolicyName].ID,
					ClusterID:    clusterId,
				}

				hostnameList = append(hostnameList, hname)
				hostList = append(hostList, *host)
			}
		}

		if foundHosts, rhErr := dbReadHosts(map[string]interface{}{"name": hostnameList}, tx); err != nil {
			return rhErr // uses default err status
		} else if len(foundHosts) > 0 {
			foundHostnames := namesOfHosts(foundHosts)
			existingHostMsg = fmt.Sprintf("on cluster update the following hosts already exist and will not be altered: %v", foundHostnames)
			if dimensionsUpdated {
				existingHostMsg = "cluster dimensions updated; " + existingHostMsg
			}
			clog.Warn().Msg(existingHostMsg)
			// filter the hostList to include only new ones
			var newHostList []Host
			var newHostnameList []string
			for _, h := range hostList {
				exists := false
				for _, n := range foundHostnames {
					if n == h.Name {
						exists = true
						break
					}
				}
				if !exists {
					newHostList = append(newHostList, h)
					newHostnameList = append(newHostnameList, h.Name)
				}
			}
			hostList = newHostList
			hostnameList = newHostnameList
		}

		if len(hostList) > 0 {
			err = dbCreateHosts(hostList, tx)
			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					status = http.StatusBadRequest
					err = fmt.Errorf("%v - one or more fields in the referenced column are duplicates", err)
				}
				return err // uses default err status
			}
		} else if dimensionsUpdated {
			// just fall through
		} else {
			status = http.StatusBadRequest
			return fmt.Errorf(existingHostMsg + " -- no new hosts created")
		}

		return nil
	}); err != nil {
		return nil, nil, status, err
	}

	for _, c := range cConfigs {
		c.storeClusterRanges()
	}

	clusters, _ = dbReadClustersTx(nil)

	if existingHostMsg != "" {
		return clusters, hostnameList, http.StatusCreated, fmt.Errorf(existingHostMsg)
	}
	return clusters, hostnameList, http.StatusCreated, nil
}

func doUpdateMotd(motdParams map[string]interface{}) (int, error) {

	cList, err := dbReadClustersTx(nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	motd, _ := motdParams["motd"].(string)
	motdUrgent := false
	if len(motd) == 0 {
		// urgent flag always false if no motd
		motdUrgent = false
	} else {
		motdUrgent, _ = motdParams["motdUrgent"].(bool)
	}

	err = dbUpdateMotdTx(cList[0].Name, motd, motdUrgent)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
