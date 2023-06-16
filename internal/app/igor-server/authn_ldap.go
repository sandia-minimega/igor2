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
	actionPrefix := "ldap group sync"

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

	if err := performDbTx(func(tx *gorm.DB) error {
		ia, _, iaErr := getIgorAdmin(tx)
		if iaErr != nil {
			return iaErr
		}
		authErr := bcrypt.CompareHashAndPassword(ia.PassHash, []byte(IgorAdmin))
		if authErr == nil {
			return fmt.Errorf("Admin pw curently set to default value")
		}
		return nil
	}); err != nil {
		logger.Warn().Msgf("%s - failed Igor Admin condition check - %v - user sync aborted", actionPrefix, err)
		return
	}

	// gather config elements
	baseDN := igor.Auth.Ldap.BaseDN
	bindDN := igor.Auth.Ldap.BindDN
	bindPW := igor.Auth.Ldap.BindPassword
	gcConf := igor.Auth.Ldap.GroupSync
	filter := "(" + gcConf.GroupFilter + ")"
	group_search_attribute := []string{gcConf.GroupAttribute}

	// build member_attributes if present
	member_attributes := []string{}
	if gcConf.GroupMemberAttributeEmail != "" {
		member_attributes = append(member_attributes, gcConf.GroupMemberAttributeEmail)
	}
	if gcConf.GroupMemberAttributeDisplayName != "" {
		member_attributes = append(member_attributes, gcConf.GroupMemberAttributeDisplayName)
	}

	// get LDAP connection
	conn, err := getLDAPConnection()
	if err != nil {
		logger.Error().Msgf(actionPrefix+" - unable to get LDAP connection during user sync - %s - user sync aborted", err.Error())
		return
	}
	defer conn.Close()

	err = conn.Bind(bindDN, bindPW)
	if err != nil {
		errLine := actionPrefix + " - Bind read-only failed: "
		logger.Error().Msgf("%s - %v - user sync aborted", errLine, err.Error())
		return
	}
	result, err := conn.Search(&ldap.SearchRequest{
		BaseDN:     baseDN,
		Scope:      ldap.ScopeWholeSubtree,
		Filter:     filter,
		Attributes: group_search_attribute,
	})

	if err != nil {
		errLine := actionPrefix + " failed: "
		logger.Error().Msgf("%s - %v - user sync aborted", errLine, err.Error())
		return
	}

	if len(result.Entries) < 1 {
		errLine := fmt.Sprintf("%s failed - no entries returned from LDAP server for given group filter \"%s\"", actionPrefix, gcConf.GroupFilter)
		logger.Error().Msgf("%v - user sync aborted", errLine)
		return
	}

	group_members := result.Entries[0].GetAttributeValues(gcConf.GroupAttribute)
	if len(group_members) == 0 {
		logger.Error().Msgf(actionPrefix + " failed - group retrieved from LDAP but contained no members - user sync aborted")
		return
	}

	// get all Igor users
	igorUsers, err := dbReadUsersTx(map[string]interface{}{})
	if err != nil {
		logger.Error().Msgf("%s failed - %v - user sync aborted", actionPrefix, err.Error())
		return
	}

	// filter out non members so we can register them
	newMembers := filterNonMembers(igorUsers, group_members)

	// stop if no new members need to be registered
	if len(newMembers) == 0 {
		logger.Debug().Msgf("No new members to create")
		return
	}

	// register each new member
	for _, member := range newMembers {
		user_filter := fmt.Sprintf("(uid=%s)", member)
		user_result, err := conn.Search(&ldap.SearchRequest{
			BaseDN:     baseDN,
			Scope:      ldap.ScopeWholeSubtree,
			Filter:     fmt.Sprintf(user_filter),
			Attributes: member_attributes,
		})
		if err != nil {
			errLine := fmt.Sprintf("%s - failed searching for user %s in LDAP: ", actionPrefix, member)
			logger.Error().Msgf("%s - %v - user sync aborted", errLine, err.Error())
			return
		}
		if len(user_result.Entries) == 0 {
			logger.Warn().Msgf(actionPrefix+" - failed to retrieve user %s from LDAP, skipping user creation", member)
			continue
		}
		entry := user_result.Entries[0]
		member_email := ""
		if gcConf.GroupMemberAttributeEmail != "" {
			if len(entry.GetAttributeValues(gcConf.GroupMemberAttributeEmail)) > 0 {
				member_email = entry.GetAttributeValues(gcConf.GroupMemberAttributeEmail)[0]
			}
		}
		if member_email == "" {
			member_email = fmt.Sprintf("%s@%s", member, igor.Email.DefaultSuffix)
		}
		member_display_name := ""
		if gcConf.GroupMemberAttributeDisplayName != "" {
			if len(entry.GetAttributeValues(gcConf.GroupMemberAttributeDisplayName)) > 0 {
				member_display_name = entry.GetAttributeValues(gcConf.GroupMemberAttributeDisplayName)[0]
			}
		}

		if user, _, err := createNewUser(member, member_email, member_display_name, &logger); err == nil {
			logger.Debug().Msg("new user creation complete")
			logger.Info().Msgf("New user %s created with group sync manager", member)
			acctCreatedMsg := makeAcctNotifyEvent(EmailAcctCreated, user)
			if acctCreatedMsg != nil {
				acctNotifyChan <- *acctCreatedMsg
			}
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
			ldapCert, err := os.ReadFile(ldapConf.TLSConfig.Cert)
			if err != nil {
				e := fmt.Sprintf("%s failed - Failed to read ad cert: %v", actionPrefix, err)
				return nil, fmt.Errorf(e)
			}
			ok := rootCA.AppendCertsFromPEM(ldapCert)
			if !ok {
				logger.Warn().Msgf("%s failed - AD cert at %s not added.", actionPrefix, ldapConf.TLSConfig.Cert)
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
		logger.Error().Msg(errLine + err.Error())
		return nil, err
	}
	// upgrade connection to TLS if configured
	if igor.Auth.Scheme != "ldaps" && ldapConf.UseTLS {
		err = c.StartTLS(tlsConfig)
		if err != nil {
			errLine := actionPrefix + " (startTLS) - failed: "
			logger.Error().Msg(errLine + err.Error())
			return nil, err
		}
	}
	return c, nil
}
