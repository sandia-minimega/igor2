// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

// doUpdateUser steps through the process of making an update to a user record.
//
// Returns:
//
//	200,nil if update was successful
//	404,error if user cannot be found
//	500,error if an internal error occurred
func doUpdateUser(username string, editParams map[string]interface{}, r *http.Request) (actionStr string, status int, err error) {

	clog := hlog.FromRequest(r)
	actionStr = "updated"
	actionUser := getUserFromContext(r)
	isSameAccount := true
	if actionUser.Name != username {
		isSameAccount = false
	}

	if email, emailOK := editParams["email"].(string); emailOK {
		editParams["email"] = strings.ToLower(email)
	}

	if fullName, fullNameOK := editParams["fullName"].(string); fullNameOK {
		editParams["FullName"] = fullName
		delete(editParams, "fullName")
	}

	reset, resetOK := editParams["reset"].(bool)
	newPassword, passOK := editParams["password"]
	oldPassword, _ := editParams["oldPassword"]
	var user *User

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		userList, guStatus, guErr := getUsers([]string{username}, true, tx)
		if guErr != nil {
			status = guStatus
			return guErr
		}
		user = &userList[0]

		if resetOK || passOK {
			if igor.Auth.Scheme == "local" || user.Name == IgorAdmin {
				if !reset {
					// password change must be performed by account owner
					if isSameAccount && passOK {
						passErr := bcrypt.CompareHashAndPassword(user.PassHash, []byte(oldPassword.(string)))
						if passErr != nil {
							clog.Warn().Msgf("attempted password change for '%s' failed - old password was incorrect", user.Name)
							status = http.StatusForbidden
							return fmt.Errorf("old password was incorrect")
						}

						if hash, hashErr := getPasswordHash(newPassword.(string)); hashErr != nil {
							return hashErr // uses default status
						} else {
							clog.Debug().Msgf("attempting to change password for '%s'", user.Name)
							editParams["pass_hash"] = hash
							actionStr = "password changed"
							delete(editParams, "password")
							delete(editParams, "oldPassword")
						}
					} else if passOK {
						status = http.StatusBadRequest
						return fmt.Errorf("must be signed in as original account to choose new password - admins must use reset password for users")
					}
				} else {
					// password reset can only be performed by admin
					if userElevated(actionUser.Name) {
						passwordDefault := igor.Auth.DefaultUserPassword
						if user.Name == IgorAdmin {
							clog.Warn().Msgf("%s password is being reset by %s", IgorAdmin, actionUser.Name)
							passwordDefault = IgorAdmin
						} else {
							clog.Info().Msgf("'%s' password is being reset by '%s'", user.Name, actionUser.Name)
						}
						if hash, hashErr := getPasswordHash(passwordDefault); hashErr != nil {
							return hashErr // uses default err status
						} else {
							editParams["pass_hash"] = hash
							actionStr = "password reset"
							delete(editParams, "reset")
						}

					} // no else case, reset by non-admin blocked by permissions
				}
			} else {
				clog.Warn().Msgf("passwords not managed by igor - authentication scheme is %s, not local", igor.Auth.Scheme)
				status = http.StatusBadRequest
				return fmt.Errorf("passwords not managed by igor (scheme = %s)", igor.Auth.Scheme)
			}
		}

		clog.Debug().Msgf("applying changes to '%s'", user.Name)
		return dbEditUser(user, editParams, tx)

	}); err == nil {
		clog.Debug().Msgf("changes to '%s' complete", username)
		status = http.StatusOK

		if resetOK && igor.Auth.Scheme == "local" {
			passResetMsg := makeAcctNotifyEvent(EmailPasswordReset, user)
			if passResetMsg != nil {
				acctNotifyChan <- *passResetMsg
			}
		}

	} else {
		actionStr = ""
	}
	return
}
