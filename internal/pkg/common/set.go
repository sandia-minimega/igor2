// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import (
	"reflect"
	"sort"
	"strings"
)

var exists = struct{}{}

// Set creates a map where string keys must be unique and values point
// to the same empty struct reference. Leading and trailing whitespace
// will be ignored when inserting/removing/searching the set keys. Whitespace
// and the empty string cannot be used as keys and trying to do so with the
// Add method will silently ignore the attempt.
type Set struct {
	vals map[string]struct{}
}

func NewSet() *Set {
	s := &Set{}
	s.vals = make(map[string]struct{})
	return s
}

func (s *Set) Add(values ...string) {
	for _, val := range values {
		val = strings.TrimSpace(val)
		if val != "" {
			s.vals[val] = exists
		}
	}
}

func (s *Set) Remove(values ...string) {
	for _, val := range values {
		val = strings.TrimSpace(val)
		delete(s.vals, val)
	}
}

func (s *Set) Contains(value string) bool {
	val := strings.TrimSpace(value)
	_, included := s.vals[val]
	return included
}

func (s *Set) ContainsAll(other *Set) bool {
	if other == nil {
		return false
	}
	for k, _ := range other.vals {
		if !s.Contains(k) {
			return false
		}
	}
	return true
}

func (s *Set) GetVals() []string {
	vals := make([]string, 0, s.Size())
	for k := range s.vals {
		vals = append(vals, k)
	}
	sort.Strings(vals)
	return vals
}

func (s *Set) Size() int {
	return len(s.vals)
}

func (s *Set) Equals(other *Set) bool {
	return reflect.DeepEqual(s.vals, other.vals)
}
