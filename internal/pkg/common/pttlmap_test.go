// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTTLMap(t *testing.T) {

	ptm := NewPassiveTtlMap(time.Second * 5)
	ptm.Put("test", true)
	assert.Equal(t, 1, ptm.Len(), "should have one member")
	time.Sleep(2 * time.Second)
	v := ptm.Get("test")
	_, ok := v.(bool)
	if !ok {
		t.FailNow()
	}
	assert.Equal(t, true, v, "should be true")
	time.Sleep(2 * time.Second)
	v = ptm.Get("test")
	_, ok = v.(bool)
	if !ok {
		t.FailNow()
	}
	assert.Equal(t, true, v, "should be true")
	time.Sleep(1 * time.Second)
	v = ptm.Get("test")
	_, ok = v.(bool)
	assert.Equal(t, nil, v, "should be nil")
	assert.Equal(t, false, ok, "should be false")
}

func TestUpdate(t *testing.T) {
	ptm := NewPassiveTtlMap(time.Second * 20)
	ptm.Put("test", 64)
	assert.Equal(t, 1, ptm.Len(), "should have one member")
	assert.Equal(t, 64, ptm.Get("test"), "value should be 64")
	ptm.Put("test", "sixty-five")
	assert.Equal(t, "sixty-five", ptm.Get("test"), "value should be 'sixty-five'")
}

func TestRemaining(t *testing.T) {
	ptm := NewPassiveTtlMap(time.Second * 5)
	ptm.Put("test", true)
	assert.Equal(t, 1, ptm.Len(), "should have one member")
	ptm.Remaining("test")
	assert.NotEqual(t, 0, ptm.Remaining("test"), "should not be 0")
	ptm.Remove("test")
	assert.Equal(t, 0, ptm.Len(), "should have no members")
	ptm.Remaining("test")
	assert.Equal(t, int64(0), ptm.Remaining("test"), "should be 0")
	v := ptm.Get("test")
	assert.Nil(t, v, "should be nil")
}

func TestNonExistentEntry(t *testing.T) {
	ptm := NewPassiveTtlMap(time.Second * 5)
	ptm.Put("test", true)
	assert.Equal(t, 1, ptm.Len(), "should have one member")
	ptm.Remaining("no-one")
	assert.Equal(t, int64(0), ptm.Remaining("no-one"), "should be 0")
	v := ptm.Get("no-one")
	assert.Nil(t, v, "should be nil")
}

func TestClear(t *testing.T) {
	ptm := NewPassiveTtlMap(time.Second * 5)
	ptm.Put("test-1", true)
	ptm.Put("test-2", true)
	ptm.Put("test-3", true)
	time.Sleep(3 * time.Second)
	ptm.Put("test-4", true)
	ptm.Put("test-5", true)
	ptm.Put("test-6", true)
	time.Sleep(2 * time.Second)
	ptm.ClearExpired()
	assert.Equal(t, 3, ptm.Len(), "should have three members")
	time.Sleep(3 * time.Second)
	ptm.Put("test-6", true)
	ptm.ClearExpired()
	assert.Equal(t, 1, ptm.Len(), "should have one member")
}
