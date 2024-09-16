// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doCreateUser creates a new Igor user. It steps through the process of checking to make sure
// there are no inherent conflicts with the db, hashing the password, creating the pug and its
// permissions, and adding the user to the 'all' group.
func doCreateUser(userParams map[string]interface{}, r *http.Request) (user *User, status int, err error) {

	clog := &logger
	if r != nil {
		clog = hlog.FromRequest(r)
	}

	username := strings.ToLower(strings.TrimSpace(userParams["name"].(string)))
	email := strings.ToLower(strings.TrimSpace(userParams["email"].(string)))
	fullName, fnOK := userParams["fullName"].(string)
	if fnOK {
		fullName = strings.TrimSpace(fullName)
	}

	status = http.StatusInternalServerError // default status, overridden if no errors
	ok := false
	if ok, status, err = checkUniqueUserAttributes(username, email); !ok {
		return nil, status, err
	}
	clog.Debug().Msgf("creating new igor user '%s'", username)
	if user, status, err = createNewUser(username, email, fullName, clog); err == nil {
		clog.Debug().Msg("new user creation complete")
		status = http.StatusCreated

		acctCreatedMsg := makeAcctNotifyEvent(EmailAcctCreated, user)
		if acctCreatedMsg != nil {
			acctNotifyChan <- *acctCreatedMsg
		}

	}
	return
}

func createNewUser(username, email, fullName string, clog *zerolog.Logger) (user *User, status int, err error) {
	status = http.StatusInternalServerError // default status, overridden at end if no errors
	err = performDbTx(func(tx *gorm.DB) error {
		clog.Debug().Msg("setting default user password")
		hash, hashErr := createPasswordHash(igor.Auth.DefaultUserPassword)
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
			Owners:        []User{*igorAdmin},
			Members:       []User{*user},
			IsLDAP:        false,
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

	})
	return user, status, err
}
