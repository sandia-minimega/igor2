package igorserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"igor2/internal/pkg/common"
	"os"
	"regexp"
	"slices"
)

func syncPreCheck() error {

	var errLine string
	// make sure cluster has been created and admin password is not default

	if ia, _, iaErr := getIgorAdminTx(); iaErr != nil {
		errLine += fmt.Sprintf("can't access DB - %v", iaErr)
	} else {
		authErr := bcrypt.CompareHashAndPassword(ia.PassHash, []byte(IgorAdmin))
		if authErr == nil {
			errLine += "admin password currently set to default value"
		}
	}

	if clusters, _, cErr := doReadClusters(map[string]interface{}{}); cErr != nil {
		if len(errLine) != 0 {
			errLine += " - "
		}
		errLine += fmt.Sprintf("error getting cluster data - %v", cErr)
	} else if (len(clusters) == 0) || clusters[0].Name == "" {
		if len(errLine) != 0 {
			errLine += " - "
		}
		errLine += "no clusters returned or empty cluster name"
	}

	if len(errLine) != 0 {
		return fmt.Errorf("LDAP sync pre-check failed: %s", errLine)
	} else {
		logger.Info().Msgf("LDAP pre-sync check passed")
		return nil
	}
}

func executeLdapUserSync() {
	if conn, err := getLDAPConnection(); err != nil {
		logger.Error().Msgf("%v", err)
		return
	} else {
		if err = syncLdapUsers(conn); err != nil {
			logger.Error().Msgf("%v", err)
		}
	}
}

func executeLdapGroupSync() {
	if conn, err := getLDAPConnection(); err != nil {
		logger.Error().Msgf("%v", err)
		return
	} else {
		groups, users, siErr := ldapGroupSyncInfo()
		if siErr != nil {
			logger.Error().Msgf("%v", siErr)
			return
		}
		if err = syncLdapGroups(conn, groups, users); err != nil {
			logger.Error().Msgf("%v", err)
		}
	}
}

func getLDAPConnection() (*ldap.Conn, error) {
	actionPrefix := "LDAP connection"

	bindDN := igor.Auth.Ldap.BindDN
	bindPW := igor.Auth.Ldap.BindPassword

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

	connErr := c.Bind(bindDN, bindPW)
	if connErr != nil {
		err = fmt.Errorf("%s - bind read-only failed - %w", actionPrefix, connErr)
		return nil, err
	}

	return c, nil
}

func executeLdapGroupCreate(group *Group) (members []User, owners []User, err error) {
	if conn, connErr := getLDAPConnection(); connErr != nil {
		err = fmt.Errorf("ldap group create failed - %v", connErr)
	} else {
		return createLdapSyncGroup(conn, group)
	}
	return
}

func createLdapSyncGroup(conn *ldap.Conn, group *Group) (members []User, owners []User, err error) {

	actionPrefix := "create LDAP-synced group"
	defer conn.Close()

	// gather config elements
	baseDN := igor.Auth.Ldap.BaseDN
	gcConf := igor.Auth.Ldap.Sync
	filter := "(cn=" + group.Name + ")"
	groupSearchAttributes := []string{gcConf.UserListAttribute}
	groupSearchAttributes = append(groupSearchAttributes, gcConf.GroupOwnerAttributes...)

	result, searchErr := conn.Search(&ldap.SearchRequest{
		BaseDN:     baseDN,
		Scope:      ldap.ScopeWholeSubtree,
		Filter:     filter,
		Attributes: groupSearchAttributes,
	})

	if searchErr != nil {
		err = fmt.Errorf("%s failed - problem retrieving LDAP search result - %v", actionPrefix, searchErr)
		return
	}

	if len(result.Entries) < 1 {
		err = fmt.Errorf("%s failed - no entries returned from LDAP server for given group name '%s'", actionPrefix, group.Name)
		return
	}

	ldapGroupMembers := result.Entries[0].GetAttributeValues(groupSearchAttributes[0])
	if len(ldapGroupMembers) == 0 {
		err = fmt.Errorf("%s failed - group '%s' retrieved from LDAP but contained no members", actionPrefix, group.Name)
		return
	}

	uid := regexp.MustCompile(`uid=(\w+),`)
	ldapGroupOwners := common.NewSet()
	for i := 1; i < len(groupSearchAttributes); i++ {
		for _, val := range result.Entries[0].GetAttributeValues(groupSearchAttributes[i]) {
			ldapGroupOwners.Add(uid.FindStringSubmatch(val)[1])
		}
	}

	if ldapGroupOwners.Size() == 0 {
		err = fmt.Errorf("%s failed - unable to find an owner for group '%s' in LDAP search results", actionPrefix, group.Name)
		return
	}

	ldapGroupMembers = append(ldapGroupMembers, ldapGroupOwners.Elements()...)

	// get all igor users
	igorUsers, ruErr := dbReadUsersTx(map[string]interface{}{})
	if ruErr != nil {
		err = fmt.Errorf("%s failed - %w", actionPrefix, ruErr)
		return
	}

	members = usersFromNames(igorUsers, ldapGroupMembers)
	owners = usersFromNames(igorUsers, ldapGroupOwners.Elements())

	if len(owners) == 0 {
		ia, _, _ := getIgorAdminTx()
		owners = append(owners, *ia)
	}

	return
}

func ldapGroupSyncInfo() ([]Group, []User, error) {

	actionPrefix := "LDAP group sync"

	ldapGroupList, rgErr := dbReadGroupsTx(map[string]interface{}{"is_ldap": true, "showMembers": true}, true)
	if rgErr != nil {
		return nil, nil, fmt.Errorf("%s failed - %w", actionPrefix, rgErr)
	} else if len(ldapGroupList) == 0 {
		return nil, nil, nil
	}

	// get all igors users
	igorUsers, ruErr := dbReadUsersTx(map[string]interface{}{})
	if ruErr != nil {
		return nil, nil, fmt.Errorf("%s failed - %w - sync aborted", actionPrefix, ruErr)
	}

	return ldapGroupList, igorUsers, nil
}

func syncLdapGroups(conn *ldap.Conn, ldapGroupList []Group, igorUsers []User) (err error) {
	actionPrefix := "LDAP group sync"
	defer conn.Close()
	if len(ldapGroupList) == 0 {
		logger.Warn().Msgf("%s - enabled but no LDAP groups are being tracked by igor - sync aborted", actionPrefix)
		return
	}

	// gather config elements
	baseDN := igor.Auth.Ldap.BaseDN
	gcConf := igor.Auth.Ldap.Sync
	groupSearchAttributes := []string{gcConf.UserListAttribute}
	groupSearchAttributes = append(groupSearchAttributes, gcConf.GroupOwnerAttributes...)
	uid := regexp.MustCompile(`uid=(\w+),`)

	for _, group := range ldapGroupList {

		result, searchErr := conn.Search(&ldap.SearchRequest{
			BaseDN:     baseDN,
			Scope:      ldap.ScopeWholeSubtree,
			Filter:     "(cn=" + group.Name + ")",
			Attributes: groupSearchAttributes,
		})

		if searchErr != nil {
			err = fmt.Errorf("%s failed - problem retrieving LDAP search result - %v", actionPrefix, searchErr)
			logger.Error().Msgf("%v", err)
			continue
		}

		if len(result.Entries) < 1 {
			err = fmt.Errorf("%s failed - no entries returned from LDAP server for given group name '%s'", actionPrefix, group.Name)
			logger.Error().Msgf("%v", err)
			continue
		}

		// get the list of group members
		ldapGroupMembers := common.NewSet()
		ldapGroupMembers.Add(result.Entries[0].GetAttributeValues(groupSearchAttributes[0])...)
		if ldapGroupMembers.Size() == 0 {
			err = fmt.Errorf("%s failed - group retrieved from LDAP but contained no members - aborted", actionPrefix)
			logger.Error().Msgf("%v", err)
			continue
		}

		// get the list of owners and delegates
		ldapGroupOwners := common.NewSet()
		for i := 1; i < len(groupSearchAttributes); i++ {
			for _, val := range result.Entries[0].GetAttributeValues(groupSearchAttributes[i]) {
				ldapGroupOwners.Add(uid.FindStringSubmatch(val)[1])
			}
		}

		requiresUpdate := false
		var addOwners, rmvOwners []string
		groupOwners := usernamesFromNames(igorUsers, ldapGroupOwners.Elements())
		currOwners := userNamesOfUsers(group.Owners)

		slices.Sort(currOwners)
		slices.Sort(groupOwners)
		if !slices.Equal(currOwners, groupOwners) {
			requiresUpdate = true
			addOwners = usernameDiff(currOwners, groupOwners)
			rmvOwners = usernameDiff(groupOwners, currOwners)
			slices.Sort(rmvOwners)
		}

		var addMembers, rmvMembers []string
		ldapGroupMembers.Add(ldapGroupOwners.Elements()...) // owners are members if in Igor but may not be according to LDAP
		groupMembers := usernamesFromNames(igorUsers, ldapGroupMembers.Elements())
		currMembers := userNamesOfUsers(group.Members)

		slices.Sort(currMembers)
		slices.Sort(groupMembers)
		if !slices.Equal(currMembers, groupMembers) {
			requiresUpdate = true
			addMembers = usernameDiff(currMembers, groupMembers)
			rmvMembers = usernameDiff(groupMembers, currMembers)
		}

		// don't change anything if igor-admin is involved
		if slices.Contains(rmvOwners, IgorAdmin) && len(addOwners) == 0 {
			rmvOwners = nil
			if len(rmvMembers) == 1 && rmvMembers[0] == IgorAdmin {
				rmvMembers = nil
			} else if len(rmvMembers) > 1 {
				if i := slices.Index(rmvMembers, IgorAdmin); i != -1 {
					rmvMembers = append(rmvMembers[:i], rmvMembers[i+1:]...)
				}
			}
		} else if len(addOwners) == 0 && len(rmvOwners) > 0 && slices.Equal(rmvOwners, currOwners) && !slices.Contains(rmvOwners, IgorAdmin) {
			// if removing all the group owners who aren't igor-admin and no replacements, igor-admin should take ownership
			addOwners = append(addOwners, IgorAdmin)
		}

		if requiresUpdate {

			changes := make(map[string]interface{}, 4)

			if len(addMembers) > 0 {
				members := usersFromNames(igorUsers, addMembers)
				if len(members) > 0 {
					changes["add"] = members
				}
			}
			if len(addOwners) > 0 {
				owners := usersFromNames(igorUsers, addOwners)
				if len(owners) > 0 {
					changes["addOwners"] = owners
				}
			}
			if len(rmvMembers) > 0 {
				changes["remove"] = usersFromNames(group.Members, rmvMembers)
			}
			if len(rmvOwners) > 0 {
				changes["rmvOwners"] = usersFromNames(group.Owners, rmvOwners)
			}

			// possible that after filtering non-igor users or igor-admin we end up with no changes
			if len(changes) == 0 {
				continue
			}

			if guErr := performDbTx(func(tx *gorm.DB) error {
				logger.Debug().Msgf("performing group update on '%s'", group.Name)
				return dbEditGroup(&group, changes, tx)
			}); guErr != nil {
				err = fmt.Errorf("problem performing group update - %w", guErr)
				logger.Error().Msgf("%v", err)
				continue
			}
		}
	}

	return
}

func syncLdapUsers(conn *ldap.Conn) error {
	actionPrefix := "LDAP user account sync"
	defer conn.Close()

	// gather config elements
	baseDN := igor.Auth.Ldap.BaseDN
	gcConf := igor.Auth.Ldap.Sync
	groupSearchAttribute := []string{gcConf.UserListAttribute}

	// build member_attributes if present
	var memberAttributes []string
	if gcConf.UserEmailAttribute != "" {
		memberAttributes = append(memberAttributes, gcConf.UserEmailAttribute)
	}
	if gcConf.UserDisplayNameAttribute != "" {
		memberAttributes = append(memberAttributes, gcConf.UserDisplayNameAttribute)
	}

	userList := common.NewSet()

	for _, groupFilter := range gcConf.GroupFilters {

		result, searchErr := conn.Search(&ldap.SearchRequest{
			BaseDN:     baseDN,
			Scope:      ldap.ScopeWholeSubtree,
			Filter:     fmt.Sprintf("(%s)", groupFilter),
			Attributes: groupSearchAttribute,
		})

		if searchErr != nil {
			logger.Error().Msgf("%s failed - problem retrieving LDAP search result - %v", actionPrefix, searchErr)
			continue
		}

		if len(result.Entries) < 1 {
			logger.Debug().Msgf("%s - no entries returned for given group filter '%s'", actionPrefix, groupFilter)
		}

		userList.Add(result.Entries[0].GetAttributeValues(gcConf.UserListAttribute)...)
	}

	if userList.Size() == 0 {
		return fmt.Errorf("%s failed - no user account names reurned given group filters", actionPrefix)
	}

	// get all Igor users
	igorUsers, ruErr := dbReadUsersTx(map[string]interface{}{})
	if ruErr != nil {
		return fmt.Errorf("%s failed - %w", actionPrefix, ruErr)
	}

	currLdapUserList := usernamesFromNames(igorUsers, userList.Elements())
	currIgorUserList := userNamesOfUsers(igorUsers)

	slices.Sort(currLdapUserList)
	slices.Sort(currIgorUserList)
	if !slices.Equal(currLdapUserList, currIgorUserList) {
		removedUsernames := usernameDiff(currLdapUserList, currIgorUserList)
		_ = removeSyncedUsers(usersFromNames(igorUsers, removedUsernames))
	}

	// filter out non-members so we can register them
	newIgorUsers := filterNonUsers(igorUsers, userList.Elements())

	// stop if no new members need to be registered
	if len(newIgorUsers) == 0 {
		logger.Debug().Msgf("no new users to create")
		return nil
	}

	// register each new member
	for _, member := range newIgorUsers {
		userFilter := fmt.Sprintf("(uid=%s)", member)
		userResult, srErr := conn.Search(&ldap.SearchRequest{
			BaseDN:     baseDN,
			Scope:      ldap.ScopeWholeSubtree,
			Filter:     userFilter,
			Attributes: memberAttributes,
		})

		if srErr != nil {
			logger.Warn().Msgf("%s failed - search for user '%s' in LDAP - %v", actionPrefix, member, srErr)
			continue
		}
		if len(userResult.Entries) == 0 {
			logger.Warn().Msgf("%s failed - no user '%s' found", actionPrefix, member)
			continue
		}

		userInfo := map[string]interface{}{"name": member}

		entry := userResult.Entries[0]
		memberEmail := ""
		if gcConf.UserEmailAttribute != "" {
			if len(entry.GetAttributeValues(gcConf.UserEmailAttribute)) > 0 {
				memberEmail = entry.GetAttributeValues(gcConf.UserEmailAttribute)[0]
				userInfo["email"] = memberEmail
			}
		}
		if memberEmail == "" {
			memberEmail = fmt.Sprintf("%s@%s", member, igor.Email.DefaultSuffix)
			userInfo["email"] = memberEmail
		}
		memberDisplayName := ""
		if gcConf.UserDisplayNameAttribute != "" {
			if len(entry.GetAttributeValues(gcConf.UserDisplayNameAttribute)) > 0 {
				memberDisplayName = entry.GetAttributeValues(gcConf.UserDisplayNameAttribute)[0]
				if len(memberDisplayName) > 0 {
					userInfo["fullName"] = memberDisplayName
				}
			}
		}

		if user, _, cuErr := doCreateUser(userInfo, nil); cuErr != nil {
			return fmt.Errorf("failed to create new user '%s' via LDAP sync manager: %v", member, cuErr)
		} else {
			logger.Info().Msgf("created new user '%s' via with LDAP sync manager", user.Name)
		}
	}

	return nil
}

func removeSyncedUsers(users []User) (err error) {

	for _, u := range users {

		if err = performDbTx(func(tx *gorm.DB) error {

			var sendEmailAlert = false
			ia, _, _ := getIgorAdmin(tx)
			groupList := u.singleOwnedGroups()

			// when user is the sole owner of a non-pug group, replace them with igor-admin
			if len(groupList) > 0 {
				sendEmailAlert = true
				changes := make(map[string]interface{})
				changes["addOwners"] = []User{*ia}
				changes["rmvOwners"] = u
				for _, g := range groupList {
					if rmErr := dbEditGroup(&g, changes, tx); rmErr != nil {
						logger.Error().Msgf("problem changing group '%s' from auto-removed owner '%s' to igor-admin: %v", g.Name, u.Name, rmErr)
					}
				}
			}

			searchByOwnerID := map[string]interface{}{"owner_id": u.ID}

			if orList, orErr := dbReadReservations(searchByOwnerID, nil, tx); orErr != nil {
				return orErr // uses default err status
			} else {
				// for any reservation they own, change ownership to igor-admin and send email alert
				if len(orList) > 0 {
					sendEmailAlert = true
					changes := make(map[string]interface{})
					changes["autoRemoveOwner"] = true
					iaPug, _ := ia.getPug()
					changes["adminPug"] = iaPug
					for _, r := range orList {
						if editErr := dbEditReservation(&r, changes, tx); editErr != nil {
							logger.Error().Msgf("problem changing reservation '%s' from auto-removed owner '%s' to igor-admin: %v", r.Name, u.Name, editErr)
						}
					}
				}
			}

			if odList, rdErr := dbReadDistros(searchByOwnerID, tx); rdErr != nil {
				return rdErr // uses default err status
			} else {
				// for any distro they own, change ownership to igor-admin and send email alert
				if len(odList) > 0 {
					sendEmailAlert = true
					changes := make(map[string]interface{})
					changes["autoRemoveOwner"] = true
					iaPug, _ := ia.getPug()
					changes["adminPug"] = iaPug
					for _, d := range odList {
						if editErr := dbEditDistro(&d, changes, tx); editErr != nil {
							logger.Error().Msgf("problem changing distro '%s' from auto-removed owner '%s' to igor-admin: %v", d.Name, u.Name, editErr)
						}
					}
				}
			}

			if sendEmailAlert {
				acctRemovedIssueMsg := makeAcctNotifyEvent(EmailAcctRemovedIssue, &u)
				if acctRemovedIssueMsg != nil {
					acctNotifyChan <- *acctRemovedIssueMsg
				}
			}

			// *** All good! let's start deleting stuff ***

			if opList, opErr := dbReadProfiles(searchByOwnerID, tx); opErr != nil {
				return opErr // uses default err status
			} else {
				for _, p := range opList {
					logger.Debug().Msgf("deleting profile '%s'", p.Name)
					if err = dbDeleteProfile(&p, tx); err != nil {
						return err // uses default err status
					}
				}
			}

			// get user PUG
			pug, pugErr := u.getPug()
			if pugErr != nil {
				return pugErr // uses default err status
			}

			// delete user PUG permissions
			logger.Debug().Msgf("deleting '%s' group permissions", pug.Name)
			if err = dbDeletePermissionsByGroup(pug, tx); err != nil {
				return err // uses default err status
			}

			// delete user PUG
			logger.Debug().Msgf("deleting '%s' group", pug.Name)
			if err = dbDeleteGroup(pug, tx); err != nil {
				return err // uses default err status
			}

			// delete the user (and their group memberships)
			logger.Debug().Msgf("deleting '%s' from the database and removing group memberships", u.Name)
			return dbDeleteUser(&u, tx)

		}); err == nil {
			logger.Debug().Msgf("user '%s' deletion complete", u.Name)
		}
	}
	return
}
