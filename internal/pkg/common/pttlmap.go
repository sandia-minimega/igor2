// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import (
	"sync"
	"time"
)

// PassiveTtlMap is a map that holds entries along with a TTL duration. A key-value pair includes a timestamp of when it
// was inserted into the map plus the TTL offset. The next time any map read method is called entries are deleted if
// their time.Now value is >= their expiration time. The Clear and ClearExpired methods can be used to forcefully clean
// up the map if needed. All map change operations are protected with a mutex.
//
// Good for situations where the stored items are short-lived or the map is not used often. This saves the need to have
// a go routine constantly checking a map that is empty most of the time.
type PassiveTtlMap struct {
	m   map[string]*expireItem
	l   sync.Mutex
	ttl time.Duration
}

type expireItem struct {
	value   interface{}
	expires time.Time
}

// NewPassiveTtlMap creates a new instance of a PassiveTtlMap. The TTL is the expiration modifier that dictates
// when an entry will be considered expired (time inserted + TTL).
func NewPassiveTtlMap(ttl time.Duration) *PassiveTtlMap {

	pTtlMap := &PassiveTtlMap{
		m:   make(map[string]*expireItem),
		ttl: ttl,
	}
	return pTtlMap
}

// Len returns the total number of entries in the map after clearing any expired entries.
func (m *PassiveTtlMap) Len() int {
	m.ClearExpired()
	return len(m.m)
}

// TTL returns the TTL value of the map.
func (m *PassiveTtlMap) TTL() time.Duration {
	return m.ttl
}

// Clear gets rid of all entries by re-initializing the inner map.
func (m *PassiveTtlMap) Clear() {
	m.l.Lock()
	m.m = make(map[string]*expireItem)
	m.l.Unlock()
}

// ClearExpired deletes all entries in the map that have reached their expiration.
func (m *PassiveTtlMap) ClearExpired() {
	m.l.Lock()
	for k := range m.m {
		if time.Until(m.m[k].expires) <= 0 {
			delete(m.m, k)
		}
	}
	m.l.Unlock()
}

// Remove deletes the entry in the map with the provided key. If k doesn't exist
// then remove is a no-op.
func (m *PassiveTtlMap) Remove(k string) {
	m.l.Lock()
	delete(m.m, k)
	m.l.Unlock()
}

// Put inserts/updates entries into the map. An update happens when the provided key
// already exists in the map. It will update the value (if different) and the entry's
// expire time to time.Now + TTL.
func (m *PassiveTtlMap) Put(k string, v interface{}) {
	m.l.Lock()
	it, ok := m.m[k]
	if !ok {
		it = &expireItem{value: v}
		m.m[k] = it
	} else {
		it.value = v
	}
	it.expires = time.Now().Add(m.ttl)
	m.l.Unlock()
}

// Get retrieves an entry from the map if its TTL hasn't been reached, otherwise
// the method returns nil while also removing the entry if present.
func (m *PassiveTtlMap) Get(k string) interface{} {
	m.l.Lock()
	defer m.l.Unlock()
	if it, ok := m.m[k]; ok {
		if time.Until(it.expires) > 0 {
			return it.value
		} else {
			delete(m.m, k)
		}
	}
	return nil
}

// Contains returns true if the entry is in the map and its TTL hasn't been reached, otherwise
// the method returns false while also removing the entry if present.
func (m *PassiveTtlMap) Contains(k string) bool {
	if m.Get(k) != nil {
		return true
	}
	return false
}

// Remaining returns the number of seconds left that entry k has before it expires. Like the Get method if the entry
// has exceeded the TTL it will delete the entry from the map. If the entry was deleted or not found, the return
// value is 0.
func (m *PassiveTtlMap) Remaining(k string) int64 {
	m.l.Lock()
	defer m.l.Unlock()
	if it, ok := m.m[k]; ok {
		remaining := time.Until(it.expires)
		if remaining > 0 {
			return remaining.Milliseconds() / 1000
		} else {
			delete(m.m, k)
		}
	}
	return 0
}
