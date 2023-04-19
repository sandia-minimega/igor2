// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHostState(t *testing.T) {

	goodHostState := "available"
	goodState := resolveHostState(goodHostState)
	assert.Equal(t, HostAvailable, goodState, "")

	badHostState := "notAvailable"
	badState := resolveHostState(badHostState)
	assert.NotEqual(t, HostAvailable, badState, "")
	assert.NotEqual(t, HostReserved, badState, "")
	assert.NotEqual(t, HostBlocked, badState, "")
	assert.NotEqual(t, HostError, badState, "")
	assert.Equal(t, HostInvalid, badState, "")

}

func TestHostState_String(t *testing.T) {
	assert.Equal(t, HostAvailable.String(), "available")
	assert.Equal(t, HostInvalid.String(), "invalid")
}
