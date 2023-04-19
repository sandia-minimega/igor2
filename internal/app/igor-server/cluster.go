// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"igor2/internal/pkg/common"
	"sort"
)

const PermClusters = "clusters"

// Cluster is a top-level aggregation of all Host resources that make up a discrete
// set of reservable units belonging to a unique named group. Individual hosts are
// referred by igor to the combination of the cluster Prefix plus the ordered number
// of the host. For example on cluster "JOTUNHEIM" that uses the prefix "jn", the
// eight host would be referred to as "jn8" which should be the hostname for that unit.
type Cluster struct {
	Base
	Name          string `gorm:"unique; notNull"`
	Prefix        string `gorm:"unique; notNull"` // The start of any given hostname on this Cluster.
	DisplayHeight int    // Height of each rack in the cluster. Only used for display purposes.
	DisplayWidth  int    // Width of each rack in the cluster. Only used for display purposes.
	Motd          string `gorm:"notNull"`
	MotdUrgent    bool   `gorm:"notNull"`
	Hosts         []Host
}

func (c *Cluster) getClusterData() common.ClusterData {

	cd := common.ClusterData{
		Name:          c.Name,
		Prefix:        c.Prefix,
		DisplayHeight: c.DisplayHeight,
		DisplayWidth:  c.DisplayWidth,
		Motd:          c.Motd,
		MotdUrgent:    c.MotdUrgent,
	}

	return cd
}

// ClusterConfig is the struct mapping of a YAML document that describes a Cluster, and some of the
// settings used by each Host that belongs to that cluster.
//
// The HostMap defines each host by node number along with the following parameters:
//
//	n:
//	 eth: (the ethernet switch identifier)
//	 ip: (the ip of the node, if static)
//	 policy: (the HostPolicy name of the node, 'default' by default)
type ClusterConfig struct {
	Prefix        string                    `yaml:"prefix"`        // The start of any given hostname on the described Cluster.
	DisplayWidth  int                       `yaml:"displayWidth"`  // Width for display purposes in CLI.
	DisplayHeight int                       `yaml:"displayHeight"` // Height for display purposes in CLI.
	HostMap       map[int]map[string]string `yaml:"hostmap"`
}

// storeClusterRanges publishes the parameters needed to do quick Cluster node range validation on reservation requests
// without the need for a database access. It will update an existing set of parameters if nodes are added or removed
// from the cluster.
func (cc *ClusterConfig) storeClusterRanges() {
	ckeys := make([]int, 0, len(cc.HostMap))
	for k := range cc.HostMap {
		ckeys = append(ckeys, k)
	}
	sort.Ints(ckeys)

	r, _ := common.NewRange(cc.Prefix, ckeys[0], ckeys[len(ckeys)-1])
	isNewRange := true
	for i, crange := range igor.ClusterRefs {
		if crange.Prefix == r.Prefix {
			igor.ClusterRefs[i] = *r
			isNewRange = false
		}
	}

	if isNewRange {
		igor.ClusterRefs = append(igor.ClusterRefs, *r)
	}
}
