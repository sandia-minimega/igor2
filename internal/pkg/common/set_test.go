// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddToSet(t *testing.T) {

	s := NewSet()
	hobbits := []string{"Frodo", "Sam", "Merry", "Pippin", "Frodo", "Merry"}
	expHobbits := []string{"Frodo", "Merry", "Pippin", "Sam"}
	s.Add(hobbits...)
	assert.Equal(t, 4, s.Size(), "should have 4 members")
	assert.Equal(t, expHobbits, s.Elements())

}
