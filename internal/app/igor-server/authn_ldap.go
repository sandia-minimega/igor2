// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog/hlog"
	"net/http"
)

// LdapAuth implements IAuth interface
type LdapAuth struct{}

// NewLdapAuth instantiates the ldap implementation of IAuth
func NewLdapAuth() IAuth {
	return &LdapAuth{}
}

func (l *LdapAuth) authenticate(r *http.Request) (*User, error) {
	ldapConf := igor.Auth.Ldap
	clog := hlog.FromRequest(r)
	actionPrefix := "ldap login"

	username, password, ok := r.BasicAuth()
	if !ok {
		errLine := actionPrefix + " failed: problem reading basic auth header"
		clog.Warn().Msgf(errLine)
		return nil, fmt.Errorf(errLine)
	}
	// verify Igor knows the user
	user, err := findUserForAuthN(username)
	if err != nil {
		clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
		return nil, err
	}

	c, err := getLDAPConnection()
	if err != nil {
		errLine := actionPrefix + " (LDAP Connection) - failed: "
		clog.Error().Msg(errLine + err.Error())
		return nil, err
	}
	defer c.Close()

	err = c.Bind(ldapConf.BindDN, ldapConf.BindPassword)
	if err != nil {
		errLine := actionPrefix + " (Bind read-only) - failed: "
		clog.Error().Msg(errLine + err.Error())
		return nil, err
	}
	filter := "(" + ldapConf.Filter + ")"
	result, err := c.Search(&ldap.SearchRequest{
		BaseDN:     ldapConf.BaseDN,
		Scope:      ldap.ScopeWholeSubtree,
		Filter:     fmt.Sprintf(filter, username),
		Attributes: ldapConf.Attributes,
	})

	if err != nil {
		errLine := actionPrefix + " (Search) - failed: "
		clog.Error().Msg(errLine + err.Error())
		return nil, err
	}

	if len(result.Entries) < 1 {
		errLine := actionPrefix + " (Search) - failed: no entries returned from LDAP server"
		clog.Error().Msg(errLine)
		return nil, err
	}

	err = c.Bind(result.Entries[0].DN, password)

	if err != nil {
		errLine := actionPrefix + " (Bind verify) - failed: "
		clog.Error().Msg(errLine + err.Error())
		if ldap.IsErrorWithCode(err, 32) || // NoSuchObject (no/bad username)
			ldap.IsErrorWithCode(err, 48) || // InappropriateAuthentication (wrong password)
			ldap.IsErrorWithCode(err, 49) || // InvalidCredentials
			ldap.IsErrorWithCode(err, 206) { // EmptyPassword
			return nil, &BadCredentialsError{msg: err.Error()}
		}
		return nil, err
	}

	return user, nil

}
