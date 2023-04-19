// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import "strconv"

const (
	HostAvailable = HostState(iota) // host is available to accept reservations
	HostReserved                    // host is running under a current reservation
	HostBlocked                     // host is blocked from being reserved (present and future)
	HostError                       // host state is in an error condition
	HostInvalid                     // placeholder for failed validation of State field (not put into DB)
)

// HostState is an enum value describing a node's current availability for reservation assignment.
// Hosts accepting reservations would be any HostState == 0.
//
//	0 = available   ; can be reserved, no active reservation
//	1 = reserved    ; running an active reservation
//	2 = blocked     ; admins have removed this node from the reservable pool
//	3 = error       ; node unresponsive, needs admin attention
type HostState int

func (s HostState) String() string {
	names := []string{"available", "reserved", "blocked", "error", "invalid"}
	i := int(s)
	switch {
	case i <= int(HostInvalid):
		return names[i]
	default:
		return strconv.Itoa(i)
	}
}

// resolveHostState maps the status string to its HostState (int) equivalent.
func resolveHostState(str string) HostState {
	names := []string{"available", "reserved", "blocked", "error"}
	for i, name := range names {
		if str == name {
			return HostState(i)
		}
	}
	return HostInvalid
}
