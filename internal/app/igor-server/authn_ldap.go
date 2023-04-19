// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog/hlog"
)

// LdapAuth implements IAuth interface
type LdapAuth struct{}

// NewLdapAuth instantiates the ldap implementation of IAuth
func NewLdapAuth() IAuth {
	return &LdapAuth{}
}

func (l *LdapAuth) authenticate(r *http.Request) (*User, error) {

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

	// prepare ldap settings
	ldapConf := igor.Auth.Ldap
	ldapServer := igor.Auth.Scheme + "://" + ldapConf.Host + ":" + ldapConf.Port

	// build tls.config with given user settings,
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if ldapConf.TLSConfig.TLSCheckPeer {
		rootCA, err := x509.SystemCertPool()
		if err != nil {
			clog.Warn().Msgf("%s failed - error loading system cert pool: %v", actionPrefix, err)
			return nil, err
		}
		if rootCA == nil {
			clog.Warn().Msgf("%s warning - couldn't locate system cert, making empty cert pool", actionPrefix)
			rootCA = x509.NewCertPool()
		}
		// If cert path was included in TLSConfig, add to rootCA
		if ldapConf.TLSConfig.Cert != "" {
			ldapCert, err := os.ReadFile(ldapConf.TLSConfig.Cert)
			if err != nil {
				clog.Warn().Msgf("%s failed - Failed to read ad cert:%v", actionPrefix, err)
				return nil, err
			}
			ok := rootCA.AppendCertsFromPEM(ldapCert)
			if !ok {
				clog.Warn().Msgf("%s failed - AD cert at %s not added.", actionPrefix, ldapConf.TLSConfig.Cert)
			}
		}
		tlsConfig = &tls.Config{
			ServerName: ldapConf.Host,
			RootCAs:    rootCA,
		}
	}

	// connect to ldap server - ldaps scheme connects using SSL, unsecured otherwise
	c, err := ldap.DialURL(ldapServer, ldap.DialWithTLSConfig(tlsConfig))
	if err != nil {
		errLine := actionPrefix + " (DialURL) - failed: "
		clog.Error().Msg(errLine + err.Error())
		return nil, err
	}
	defer c.Close()

	// upgrade connection to TLS if configured
	if igor.Auth.Scheme != "ldaps" && ldapConf.UseTLS {
		err = c.StartTLS(tlsConfig)
		if err != nil {
			errLine := actionPrefix + " (startTLS) - failed: "
			clog.Error().Msg(errLine + err.Error())
			return nil, err
		}
	}

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
