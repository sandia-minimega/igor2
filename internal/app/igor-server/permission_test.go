// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

// This code is a port to Golang of classes from Apache Shiro, originally written in Java.
// https://github.com/apache/shiro/blob/1.7.x/core/src/test/java/org/apache/shiro/authz/permission/WildcardPermissionTest.java

package igorserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlankPermission(t *testing.T) {
	_, err := NewPermission("   	")
	assert.Error(t, err, "Should have thrown error")
}

func TestEmptyPermission(t *testing.T) {
	_, err := NewPermission("")
	assert.Error(t, err, "Should have thrown error")
}

func TestOnlyDelimiters(t *testing.T) {
	_, err := NewPermission("::,,::,:")
	assert.Error(t, err, "Should have thrown error")
}

func TestPermissionLists(t *testing.T) {

	var p1, p2, p3 *Permission

	p1, _ = NewPermission("one,two")
	p2, _ = NewPermission("one")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))

	p1, _ = NewPermission("one,two,three")
	p2, _ = NewPermission("one,three")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))

	p1, _ = NewPermission("one,two:one,two,three")
	p2, _ = NewPermission("one:three")
	p3, _ = NewPermission("one:two,three")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))
	assert.True(t, p1.Implies(p3))
	assert.False(t, p2.Implies(p3))
	assert.True(t, p3.Implies(p2))

	p1, _ = NewPermission("one,two,three:one,two,three:one,two")
	p2, _ = NewPermission("one:three:two")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))

	p1, _ = NewPermission("one")
	p2, _ = NewPermission("one:two,three,four")
	p3, _ = NewPermission("one:two,three,four:five:six:seven")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.False(t, p2.Implies(p1))
	assert.False(t, p3.Implies(p1))
	assert.True(t, p2.Implies(p3))

	p1, _ = NewPermission("res:res1:*")
	p2, _ = NewPermission("res:res1:profile,power,extend,vlan")
	p3, _ = NewPermission("res:res1:profile,group,extend")
	assert.True(t, p1.Implies(p3))
	assert.False(t, p2.Implies(p3))

}

func TestListDifferentOrder(t *testing.T) {
	p6, _ := NewPermission("one,two:three,four")
	p6DiffOrder, _ := NewPermission("two,one:four,three")
	assert.True(t, p6.Equals(p6DiffOrder))
	assert.True(t, p6.Implies(p6DiffOrder))
}

func TestPermissionEquality(t *testing.T) {
	var p1, p2, p3, p4, p5, p6, p7 *Permission

	p1, _ = NewPermission("*")
	p2, _ = NewPermission("*:*")
	p3, _ = NewPermission("*:*:*")
	p4, _ = NewPermission("one,*:*,four")
	p5, _ = NewPermission("*:*:*:*")
	assert.True(t, p1.Equals(p2))
	assert.True(t, p1.Equals(p3))
	assert.True(t, p1.Equals(p4))
	assert.True(t, p1.Equals(p5))
	assert.True(t, p4.Equals(p1))

	p6, _ = NewPermission("hobbit")
	p7, _ = NewPermission("hobbit:pippin")

	assert.True(t, p6.Equals(p7))
	assert.False(t, p7.Equals(p6))
	assert.False(t, p1.Equals(p7))
}

func TestWildcards(t *testing.T) {
	var p1, p2, p3, p4, p5, p6, p7, p8, p9 *Permission

	p1, _ = NewPermission("*")
	p2, _ = NewPermission("one")
	p3, _ = NewPermission("one:two")
	p4, _ = NewPermission("one,two:three,four")
	p5, _ = NewPermission("one,two:three,four,five:six:seven,eight")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))

	p1, _ = NewPermission("newsletter:*")
	p2, _ = NewPermission("newsletter:read")
	p3, _ = NewPermission("newsletter:read,write")
	p4, _ = NewPermission("newsletter:*")
	p5, _ = NewPermission("newsletter:*:*")
	p6, _ = NewPermission("newsletter:*:read")
	p7, _ = NewPermission("newsletter:write:*")
	p8, _ = NewPermission("newsletter:read,write:*")
	p9, _ = NewPermission("newsletter")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))
	assert.True(t, p1.Implies(p9))

	p1, _ = NewPermission("newsletter:*:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))
	assert.True(t, p1.Implies(p9))

	p1, _ = NewPermission("newsletter:*:*:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))
	assert.True(t, p1.Implies(p9))

	p1, _ = NewPermission("newsletter")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))
	assert.True(t, p1.Implies(p9))

	p1, _ = NewPermission("newsletter:*:read")
	p2, _ = NewPermission("newsletter:123:read")
	p3, _ = NewPermission("newsletter:123,456:read,write")
	p4, _ = NewPermission("newsletter:read")
	p5, _ = NewPermission("newsletter:read,write")
	p6, _ = NewPermission("newsletter:123:read:write")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p1.Implies(p3))
	assert.False(t, p1.Implies(p4))
	assert.False(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))

	p1, _ = NewPermission("newsletter:*:read:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p6))
}

func TestWildcardLeftTermination(t *testing.T) {
	var p1, p2, p3, p4 *Permission

	p1, _ = NewPermission("one")
	p2, _ = NewPermission("one:*")
	p3, _ = NewPermission("one:*:*")
	p4, _ = NewPermission("one:read")

	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))

	assert.True(t, p2.Implies(p1))
	assert.True(t, p2.Implies(p3))
	assert.True(t, p2.Implies(p4))

	assert.True(t, p3.Implies(p1))
	assert.True(t, p3.Implies(p2))
	assert.True(t, p3.Implies(p4))

	assert.False(t, p4.Implies(p1))
	assert.False(t, p4.Implies(p2))
	assert.False(t, p4.Implies(p3))
}

func TestString(t *testing.T) {
	p1, _ := NewPermission("*")
	p2, _ := NewPermission("one")
	p3, _ := NewPermission("one:two")
	p4, _ := NewPermission("one,two:three,four")
	p5, _ := NewPermission("one,two:three,four,five:six:seven,eight")

	assert.True(t, p1.String() == "*")
	newAP, _ := NewPermission("*")
	assert.True(t, p1.Equals(newAP))

	assert.True(t, p2.String() == "one")
	newP2, _ := NewPermission(p2.String())
	assert.True(t, p2.Equals(newP2))

	assert.True(t, p3.String() == "one:two")
	newP3, _ := NewPermission(p3.String())
	assert.True(t, p3.Equals(newP3))

	assert.True(t, p4.String() == "one,two:four,three")
	newP4, _ := NewPermission(p4.String())
	assert.True(t, p4.Equals(newP4))

	assert.True(t, p5.String() == "one,two:five,four,three:six:eight,seven")
	newP5, _ := NewPermission(p5.String())
	assert.True(t, p5.Equals(newP5))
}
