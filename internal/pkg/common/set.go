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
	elements map[string]struct{}
}

func NewSet() *Set {
	s := &Set{}
	s.elements = make(map[string]struct{})
	return s
}

func (s *Set) Add(elems ...string) {
	for _, elem := range elems {
		elem = strings.TrimSpace(elem)
		if elem != "" {
			s.elements[elem] = exists
		}
	}
}

func (s *Set) Remove(elems ...string) {
	for _, elem := range elems {
		elem = strings.TrimSpace(elem)
		delete(s.elements, elem)
	}
}

func (s *Set) Contains(element string) bool {
	elem := strings.TrimSpace(element)
	_, included := s.elements[elem]
	return included
}

func (s *Set) ContainsAll(other *Set) bool {
	if other == nil {
		return false
	}
	for k, _ := range other.elements {
		if !s.Contains(k) {
			return false
		}
	}
	return true
}

// Elements returns all values in the set as a string slice.
func (s *Set) Elements() []string {
	elems := make([]string, 0, s.Size())
	for k := range s.elements {
		elems = append(elems, k)
	}
	sort.Strings(elems)
	return elems
}

func (s *Set) Size() int {
	return len(s.elements)
}

func (s *Set) Equals(other *Set) bool {
	return reflect.DeepEqual(s.elements, other.elements)
}
