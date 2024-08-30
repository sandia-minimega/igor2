// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"strings"
)

const (
	IgorAdmin = "igor-admin"
	PermUsers = "users"
)

// User stores information about an igor user.
type User struct {
	Base
	Name     string `gorm:"unique; notNull"`
	FullName string
	Email    string `gorm:"unique"`
	PassHash []byte
	Groups   []Group `gorm:"many2many:groups_users;"`
}

func (u *User) getUserData(actionUser *User) *common.UserData {

	var email string
	var groups []string

	if actionUser.ID == u.ID || userElevated(actionUser.Name) {
		email = u.Email
		if len(u.Groups) > 0 {
			groupNames := groupNamesOfGroups(u.Groups)
			for _, gn := range groupNames {
				if !(strings.HasPrefix(gn, GroupUserPrefix) || gn == GroupAll) {
					groups = append(groups, gn)
				}
			}
		}
	}

	var userData = &common.UserData{
		Name:     u.Name,
		FullName: u.FullName,
		Email:    email,
		Groups:   groups,
		JoinDate: u.CreatedAt.Unix(),
	}

	return userData
}

// UserAuthInfo contains the array of Groups the user owns/belongs to and the array of Permissions
// the user is granted personally and through group membership.
type UserAuthInfo struct {
	Groups      []Group
	Permissions []Permission
}

// The user's authorization info contains the list of groups they belong to and the list
// of permissions granted via those groups.
func (u *User) getAuthzInfo() (*UserAuthInfo, error) {

	userAuthInfo := &UserAuthInfo{
		Groups: u.Groups,
	}

	var permList []Permission

	// Members in the admin group who have been added to ElevateMap get admin
	// privileges, otherwise treat as a normal user.
	if userElevated(u.Name) {
		p, _ := NewPermission(PermWildcardToken)
		permList = append(permList, *p)
	} else {
		perms, err := dbGetPermissionsByGroupTx(u.Groups)
		if err != nil {
			return nil, err
		}
		permList = perms
	}

	userAuthInfo.Permissions = permList

	return userAuthInfo, nil
}

// getPug returns the user's private group. This method assumes the Groups field
// was loaded.
func (u *User) getPug() (*Group, error) {
	if len(u.Groups) == 0 {
		return nil, fmt.Errorf("getPug: user struct had nothing in groups field")
	}
	for _, g := range u.Groups {
		if g.Name == GroupUserPrefix+u.Name {
			return &g, nil
		}
	}
	return nil, fmt.Errorf("getPug: user struct groups field had no pug")
}

// getPugID returns the user's private group ID.
func (u *User) getPugID() (int, error) {
	pug, err := u.getPug()
	if err != nil {
		return -1, err
	}
	return pug.ID, nil
}

// isMemberOfGroup determines whether the user is a member of the given group.
func (u *User) isMemberOfGroup(g *Group) bool {
	if g.Name == GroupAll || g.Name == GroupUserPrefix+u.Name {
		return true
	}
	return groupSliceContains(u.Groups, g.Name)
}

// isMemberOfAnyGroup determines whether the user belongs to any of the groups in the slice
func (u *User) isMemberOfAnyGroup(gs []Group) bool {
	for _, g := range gs {
		if u.isMemberOfGroup(&g) {
			return true
		}
	}
	return false
}

// isMemberOfGroups determines whether the user is a member of the given slice of groups
func (u *User) isMemberOfGroups(gs []Group) (bool, string) {
	for _, g := range gs {
		if !u.isMemberOfGroup(&g) {
			return false, g.Name
		}
	}
	return true, ""
}

// singleOwnedGroups returns the list of groups the user solely owns. Note this doesn't include the
// user's pug since that is a system-created group owned by IgorAdmin.
func (u *User) singleOwnedGroups() (owned []Group) {
	for _, g := range u.Groups {
		if len(g.Owners) == 1 && g.Owners[0].ID == u.ID {
			owned = append(owned, g)
		}
	}
	return
}

func (u *User) sharedOwnedGroups() (owned []Group) {
	for _, g := range u.Groups {
		if len(g.Owners) > 1 {
			owned = append(owned, g)
		}
	}
	return
}

// IsPermitted iterates through all Permissions granted to the user and evaluates them against the Permission
// needed to successfully access the resource referenced by the HTTP request. When any granted permission implies the
// requested permission it returns true, otherwise it returns false.
func (uai *UserAuthInfo) IsPermitted(p *Permission) bool {

	if uai.Permissions != nil && len(uai.Permissions) > 0 {
		for _, perm := range uai.Permissions {
			if perm.Implies(p) {
				return true
			}
		}
	}
	return false
}
