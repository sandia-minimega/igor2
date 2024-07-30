// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"igor2/internal/pkg/common"

	zl "github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

func assembleYamlOutput(clusters []Cluster) ([]byte, error) {

	ccs := make(map[string]ClusterConfig)

	for _, c := range clusters {
		cc := &ClusterConfig{}
		cc.Prefix = c.Prefix
		cc.DisplayWidth = c.DisplayWidth
		cc.DisplayHeight = c.DisplayHeight
		cc.HostMap = make(map[int]map[string]string)
		for _, h := range c.Hosts {
			tempMap := make(map[string]string)
			tempMap["mac"] = h.Mac
			tempMap["hostname"] = h.HostName
			tempMap["eth"] = h.Eth
			tempMap["policy"] = h.HostPolicy.Name
			tempMap["ip"] = h.IP
			cc.HostMap[h.SequenceID] = tempMap
		}
		ccs[c.Name] = *cc
	}

	yDoc, err := yaml.Marshal(&ccs)
	if err != nil {
		return nil, err
	}
	return yDoc, nil
}

func updateClusterConfigFile(yDoc []byte, clog *zl.Logger) (string, error) {

	// anonymous function backs up file and writes new one with original name
	moveAndDump := func(filepath string, configDoc string) error {
		if _, pathErr := os.Stat(filepath); pathErr == nil {
			moveFilePath := filepath + "." + time.Now().Format(common.DateTimeFilenameFormat) + ".bak"
			clog.Info().Msgf("moving old cluster config file to %s", moveFilePath)

			if mvErr := os.Rename(filepath, moveFilePath); mvErr != nil {
				return mvErr
			} else {
				if f, fileErr := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); fileErr != nil {
					return fileErr
				} else {
					_, fileErr = f.WriteString(configDoc)
					if fileErr != nil {
						return fileErr
					}
					f.Close()
				}
			}
		} else {
			return pathErr
		}
		return nil
	}

	clusterConfigLocHome := filepath.Join(igor.IgorHome, "conf", IgorClusterConfDefault)

	if dumpErr := moveAndDump(IgorClusterConfPathDefault, string(yDoc)); dumpErr != nil {
		var pathErr *os.PathError
		if errors.As(dumpErr, &pathErr) && pathErr.Op == "stat" {
			// couldn't locate cluster conf file in /etc/igor
			clog.Warn().Msgf("%v - trying %s", pathErr, clusterConfigLocHome)
		}

		if dumpErr = moveAndDump(clusterConfigLocHome, string(yDoc)); dumpErr != nil {
			var pathErr *os.PathError
			if errors.As(dumpErr, &pathErr) && pathErr.Op == "stat" {
				return "", fmt.Errorf("no config found at %s or %s - %w", IgorClusterConfPathDefault, clusterConfigLocHome, pathErr)
			}
		} else {
			return clusterConfigLocHome, nil
		}
	}

	return IgorClusterConfPathDefault, nil
}
