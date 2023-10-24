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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

func syncUsers() {
	actionPrefix := "ldap user sync"

	// make sure cluster has been created and admin password is not default
	clusters, _, err := doReadClusters(map[string]interface{}{})
	if err != nil {
		logger.Error().Msgf("%s - error getting cluster data - %v - user sync aborted", actionPrefix, err)
		return
	}
	if (len(clusters) == 0) || clusters[0].Name == "" {
		logger.Warn().Msgf("%s - no clusters returned or empty cluster name, skipping user sync", actionPrefix)
		return
	}

	if err = performDbTx(func(tx *gorm.DB) error {
		ia, _, iaErr := getIgorAdmin(tx)
		if iaErr != nil {
			return iaErr
		}
		authErr := bcrypt.CompareHashAndPassword(ia.PassHash, []byte(IgorAdmin))
		if authErr == nil {
			return fmt.Errorf("admin password currently set to default value!")
		}
		return nil
	}); err != nil {
		logger.Warn().Msgf("%s - failed igor admin condition check - %v - user sync aborted", actionPrefix, err)
		return
	}

	// gather config elements
	baseDN := igor.Auth.Ldap.BaseDN
	bindDN := igor.Auth.Ldap.BindDN
	bindPW := igor.Auth.Ldap.BindPassword
	gcConf := igor.Auth.Ldap.GroupSync
	filter := "(" + gcConf.GroupFilter + ")"
	groupSearchAttribute := []string{gcConf.GroupAttribute}

	// build member_attributes if present
	memberAttributes := []string{}
	if gcConf.GroupMemberAttributeEmail != "" {
		memberAttributes = append(memberAttributes, gcConf.GroupMemberAttributeEmail)
	}
	if gcConf.GroupMemberAttributeDisplayName != "" {
		memberAttributes = append(memberAttributes, gcConf.GroupMemberAttributeDisplayName)
	}

	// get LDAP connection
	conn, connErr := getLDAPConnection()
	if connErr != nil {
		logger.Error().Msgf("%s failed - unable to get LDAP connection during user sync - %w - user sync aborted", actionPrefix, err.Error())
		return
	}
	defer conn.Close()

	connErr = conn.Bind(bindDN, bindPW)
	if connErr != nil {
		errLine := actionPrefix + " - bind read-only failed: "
		logger.Error().Msgf("%s - %w - user sync aborted", errLine, err.Error())
		return
	}
	result, searchErr := conn.Search(&ldap.SearchRequest{
		BaseDN:     baseDN,
		Scope:      ldap.ScopeWholeSubtree,
		Filter:     filter,
		Attributes: groupSearchAttribute,
	})

	if searchErr != nil {
		errLine := actionPrefix + " failed: "
		logger.Error().Msgf("%s - %v - user sync aborted", errLine, err.Error())
		return
	}

	if len(result.Entries) < 1 {
		errLine := fmt.Sprintf("%s failed - no entries returned from LDAP server for given group filter \"%s\"", actionPrefix, gcConf.GroupFilter)
		logger.Error().Msgf("%v - user sync aborted", errLine)
		return
	}

	groupMembers := result.Entries[0].GetAttributeValues(gcConf.GroupAttribute)
	if len(groupMembers) == 0 {
		logger.Error().Msgf(actionPrefix + " failed - group retrieved from LDAP but contained no members - user sync aborted")
		return
	}

	// get all Igor users
	igorUsers, ruErr := dbReadUsersTx(map[string]interface{}{})
	if ruErr != nil {
		logger.Error().Msgf("%s failed - %v - user sync aborted", actionPrefix, err.Error())
		return
	}

	// filter out non-members so we can register them
	newMembers := filterNonMembers(igorUsers, groupMembers)

	// stop if no new members need to be registered
	if len(newMembers) == 0 {
		logger.Debug().Msgf("No new members to create")
		return
	}

	// register each new member
	for _, member := range newMembers {
		userFilter := fmt.Sprintf("(uid=%s)", member)
		userResult, srErr := conn.Search(&ldap.SearchRequest{
			BaseDN:     baseDN,
			Scope:      ldap.ScopeWholeSubtree,
			Filter:     fmt.Sprintf(userFilter),
			Attributes: memberAttributes,
		})
		if srErr != nil {
			errLine := fmt.Sprintf("%s - failed searching for user %s in LDAP: ", actionPrefix, member)
			logger.Error().Msgf("%s - %v - user sync aborted", errLine, err.Error())
			return
		}
		if len(userResult.Entries) == 0 {
			logger.Warn().Msgf(actionPrefix+" - failed to retrieve user %s from LDAP, skipping user creation", member)
			continue
		}
		entry := userResult.Entries[0]
		memberEmail := ""
		if gcConf.GroupMemberAttributeEmail != "" {
			if len(entry.GetAttributeValues(gcConf.GroupMemberAttributeEmail)) > 0 {
				memberEmail = entry.GetAttributeValues(gcConf.GroupMemberAttributeEmail)[0]
			}
		}
		if memberEmail == "" {
			memberEmail = fmt.Sprintf("%s@%s", member, igor.Email.DefaultSuffix)
		}
		memberDisplayName := ""
		if gcConf.GroupMemberAttributeDisplayName != "" {
			if len(entry.GetAttributeValues(gcConf.GroupMemberAttributeDisplayName)) > 0 {
				memberDisplayName = entry.GetAttributeValues(gcConf.GroupMemberAttributeDisplayName)[0]
			}
		}

		if user, _, cuErr := doCreateUser(map[string]interface{}{"name": member, "email": memberEmail, "fullName": memberDisplayName}, nil); cuErr != nil {
			logger.Error().Msgf("failed to create new user '%s' via group sync manager: %v", member, cuErr)
		} else {
			logger.Info().Msgf("created new user '%s' via with group sync manager", user.Name)
		}
	}
}

func getLDAPConnection() (*ldap.Conn, error) {
	actionPrefix := "ldap connection"
	// prepare ldap settings
	ldapConf := igor.Auth.Ldap
	ldapServer := igor.Auth.Scheme + "://" + ldapConf.Host + ":" + ldapConf.Port

	// build tls.config with given user settings,
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if ldapConf.TLSConfig.TLSCheckPeer {
		rootCA, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		if rootCA == nil {
			logger.Warn().Msgf("%s warning - couldn't locate system cert, making empty cert pool", actionPrefix)
			rootCA = x509.NewCertPool()
		}
		// If cert path was included in TLSConfig, add to rootCA
		if ldapConf.TLSConfig.Cert != "" {
			ldapCert, rfErr := os.ReadFile(ldapConf.TLSConfig.Cert)
			if rfErr != nil {
				e := fmt.Sprintf("%s failed - failed to read added cert: %v", actionPrefix, rfErr)
				return nil, fmt.Errorf(e)
			}
			ok := rootCA.AppendCertsFromPEM(ldapCert)
			if !ok {
				logger.Warn().Msgf("%s failed - AD cert at %s not added.", actionPrefix, ldapConf.TLSConfig.Cert)
			}
		}
		tlsConfig.InsecureSkipVerify = false
		tlsConfig.ServerName = ldapConf.Host
		tlsConfig.RootCAs = rootCA
	}

	// connect to ldap server - ldaps scheme connects using SSL, unsecured otherwise
	c, err := ldap.DialURL(ldapServer, ldap.DialWithTLSConfig(tlsConfig))
	if err != nil {
		return nil, fmt.Errorf("%s (DialURL) - failed: %v", actionPrefix, err)
	}
	// upgrade connection to TLS if configured
	if igor.Auth.Scheme != "ldaps" && ldapConf.UseTLS {
		err = c.StartTLS(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("%s (startTLS) - failed: %v", actionPrefix, err)
		}
	}
	return c, nil
}
