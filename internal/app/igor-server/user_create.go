// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

// doCreateUser creates a new Igor user. It steps through the process of checking to make sure
// there are no inherent conflicts with the db, hashing the password, creating the pug and its
// permissions, and adding the user to the 'all' group.
func doCreateUser(userParams map[string]interface{}, r *http.Request) (user *User, status int, err error) {

	clog := hlog.FromRequest(r)
	username := strings.ToLower(strings.TrimSpace(userParams["name"].(string)))
	email := strings.ToLower(strings.TrimSpace(userParams["email"].(string)))
	fullName, fnOK := userParams["fullName"].(string)
	if fnOK {
		fullName = strings.TrimSpace(fullName)
	}
	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {
		clog.Debug().Msgf("creating new igor user '%s'", username)
		exists, ueErr := userExists(username, tx)
		if ueErr != nil {
			return ueErr // uses default err status
		}
		if exists {
			status = http.StatusConflict
			return fmt.Errorf("user '%s' already exists", username)
		} else {
			emailList, emErr := dbReadUsers(map[string]interface{}{"email": email}, tx)
			if emErr != nil {
				return emErr // uses default err status
			}
			if len(emailList) > 0 {
				status = http.StatusConflict
				return fmt.Errorf("email '%s' already used by '%s'", email, emailList[0].Name)
			}
		}

		clog.Debug().Msg("setting default user password")
		hash, hashErr := getPasswordHash(igor.Auth.DefaultUserPassword)
		if hashErr != nil {
			return hashErr // uses default err status
		}

		user = &User{
			Name:     username,
			Email:    email,
			PassHash: hash,
			FullName: fullName,
		}

		// create the actual user account
		clog.Debug().Msgf("creating user entry '%s'", username)
		err = dbCreateUser(user, tx)
		if err != nil {
			return err // uses default err status
		}

		p := &Permission{
			Fact: NewPermissionString(PermUsers, username, PermEditAction, "email,password,fullName"),
		}

		igorAdmin, iaStatus, iaErr := getIgorAdmin(tx)
		if iaErr != nil {
			status = iaStatus
			return iaErr
		}

		uGroup := &Group{
			Name:          GroupUserPrefix + username,
			Description:   username + " private group",
			IsUserPrivate: true,
			Permissions:   []Permission{*p},
			OwnerID:       igorAdmin.ID,
			Members:       []User{*user},
		}

		clog.Debug().Msgf("creating private user group for '%s'", username)
		if err = dbCreateGroup(uGroup, true, tx); err != nil {
			return err // uses default err status
		}

		gAll, gaStatus, gaErr := getAllGroup(tx)
		if gaErr != nil {
			status = gaStatus
			return gaErr
		}

		// add the new user to the 'all' group
		clog.Debug().Msgf("adding new user '%s' to the '%s' group", username, GroupAll)
		editParams := map[string]interface{}{"add": []User{*user}}
		return dbEditGroup(gAll, editParams, tx) // uses default err status

	}); err == nil {
		clog.Debug().Msg("new user creation complete")
		status = http.StatusCreated

		acctCreatedMsg := makeAcctNotifyEvent(EmailAcctCreated, user)
		if acctCreatedMsg != nil {
			acctNotifyChan <- *acctCreatedMsg
		}

	}
	return
}
