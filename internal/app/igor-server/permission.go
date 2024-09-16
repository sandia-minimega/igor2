// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

// This code is a port to Golang of a class from Apache Shiro, originally written in Java and made available under the
// Apache 2.0 License.
// https://github.com/apache/shiro/blob/main/core/src/main/java/org/apache/shiro/authz/permission/WildcardPermission.java
// Please see the NOTICES section for license information.

package igorserver

import (
	"bytes"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"igor2/internal/pkg/common"
)

const (
	PermWildcardToken = "*"
	PermDividerToken  = ":"
	PermSubpartToken  = ","
	PermCreateAction  = "create"
	PermViewAction    = "view"
	PermEditAction    = "edit"
	PermDeleteAction  = "delete"
	PermPowerAction   = "power"
)

// Permission is a piece of data that explains what kind of access a user has to a given resource or set of resources.
// It can be represented by a simple string called a Fact. In operation a fact is represented as an array of Sets that
// can be used for comparison against another Permission to determine if one implies the other.
type Permission struct {
	Base
	GroupID int          `gorm:"notNull; uniqueIndex:idx_group_fact"`
	Fact    string       `gorm:"notNull; uniqueIndex:idx_group_fact"` // The string form of a Permission.
	Parts   []common.Set `gorm:"-"`                                   // This can't be stored in the DB but is reconstructed from the fact.
}

// NewPermission creates a Permission struct from a properly formatted permission string.
func NewPermission(permString string) (*Permission, error) {
	p := &Permission{}
	err := p.setParts(permString)
	if err != nil {
		return nil, err
	}
	p.Fact = p.String()
	return p, nil
}

// NewPermissionString creates the string version of a Permission from distinct fields. If the
// fields represent multiple resources it must use the PermSubpartToken as internal dividers. Examples:
//
//	NewPermissionString("profiles", "experiment-1", "edit", "name,owner")
//	 --> "profiles:experiment-1:edit:name,owner"
//
//	NewPermissionString(PermHosts, "kn1" + PermSubpartToken + "kn2", PermViewAction)
//	 --> "hosts:kn1,kn2:view"
func NewPermissionString(fields ...string) string {
	var permString string

	// determine if all fields contain wildcards
	isAll := true
	for _, f := range fields {
		if !strings.Contains(f, PermWildcardToken) {
			isAll = false
			break
		}
	}
	if isAll {
		return PermWildcardToken
	}
	for i := 0; i < len(fields)-1; i++ {
		permString += fields[i] + PermDividerToken
	}
	permString += fields[len(fields)-1]
	return permString
}

// AfterFind populates the Parts field of a Permission struct after it is fetched from the DB, but
// before it is populated in the DB call result.
func (p *Permission) AfterFind(_ *gorm.DB) (err error) {
	return p.setParts(p.Fact)
}

// Implies returns true if this current instance implies all the functionality and/or resource access
// described by the specified Permission argument, false otherwise.
//
// That is, this current instance must be exactly equal to or a superset of the functionality
// and/or resource access described by the given Permission argument. Yet another way of saying this
// would be:
//
// If "permission1 implies permission2", then any user granted permission1 would have ability
// greater than or equal to that defined by permission2.
func (p *Permission) Implies(other *Permission) bool {

	otherParts := other.getParts()
	if len(otherParts) == 0 {
		return false
	}

	i := 0
	for _, otherPart := range otherParts {
		// If this permission has less parts than the other permission, everything after the number of parts contained
		// in this permission is automatically implied, so return true
		if len(p.Parts)-1 < i {
			return true
		} else {
			part := p.Parts[i]
			if !part.Contains(PermWildcardToken) && !part.ContainsAll(&otherPart) {
				return false
			}
			i++
		}
	}

	// If this permission has more parts than the other one, only imply it if all the other parts are wildcards
	for ; i < len(p.Parts); i++ {
		part := p.Parts[i]
		if !part.Contains(PermWildcardToken) {
			return false
		}
	}

	return true
}

func (p *Permission) getParts() []common.Set {
	return p.Parts
}

func (p *Permission) setParts(permission string) error {

	permission = strings.TrimSpace(permission)

	if len(permission) == 0 {
		return fmt.Errorf("permission string cannot be empty")
	}

	parts := strings.Split(permission, PermDividerToken)

	// if this is a long string of wildcards, just shorten it
	allWildcards := true
	for _, p := range parts {
		if !strings.Contains(p, PermWildcardToken) {
			allWildcards = false
			break
		}
	}
	if allWildcards {
		parts = []string{PermWildcardToken}
	}

	p.Parts = make([]common.Set, 0, 20)

	logger.Trace().Msgf("Building permission string using: %v", parts)

	for _, part := range parts {
		// if the part contains a wildcard token, the whole part is a wildcard token
		if strings.Contains(part, PermWildcardToken) {
			part = PermWildcardToken
		}
		partsSet := common.NewSet()
		subParts := strings.Split(part, PermSubpartToken)
		partsSet.Add(subParts...)
		if partsSet.Size() == 0 {
			return fmt.Errorf("permission string cannot contain parts with only dividers")
		}
		p.Parts = append(p.Parts, *partsSet)
	}

	if len(p.Parts) == 0 {
		return fmt.Errorf("permission string cannot contain parts with only dividers")
	}

	return nil
}

// String prints out a string version of the permission. Due to the underlying Set data structure
// of a permission, the equality of the string it produces against the original string will
// not necessarily match
//
//	p1 := "b,c,a:y,x,z"
//	wp1 := NewWildCardPermission("b,c,a:y,x,z")
//	p1 == wp1.String()  <-- fail
//
// because the underlying Sets will print out in sorted string order.
//
//	wp1.String() == "a,b,c:x,y,z"
//
// However this does NOT break the Equals method, so ...
//
//	wp2 := NewWildCardPermission("c,a,b:z,y,x")
//	wp1.String() == wp2.String()  <-- pass
func (p *Permission) String() string {

	var buffer bytes.Buffer
	for _, set := range p.Parts {
		if buffer.Len() > 0 {
			buffer.Write([]byte(PermDividerToken))
		}
		for i, k := range set.Elements() {
			buffer.Write([]byte(k))
			if i < set.Size()-1 {
				buffer.Write([]byte(PermSubpartToken))
			}
		}
	}
	return buffer.String()
}

// Equals tests equality of two permissions objects, meaning they reference
// the same fields even if the subparts are in different order. For example,
// the permission "one,two:three,four" is equal to "two,one:four,three"
func (p *Permission) Equals(other *Permission) bool {

	for _, s := range p.Parts {
		foundMatch := false
		for _, so := range other.Parts {
			if s.Equals(&so) {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return false
		}
	}

	return true
}
