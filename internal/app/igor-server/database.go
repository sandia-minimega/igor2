// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"igor2/internal/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	dbAccess sync.Mutex

	stdGormConfig = &gorm.Config{
		Logger:      gLogger,
		PrepareStmt: true,
	}
)

// IGormDb is an interface that provides the database we are connecting to via GORM.
type IGormDb interface {
	// GetDB returns a gorm DB handle to the database instance
	GetDB() *gorm.DB
}

type GormBackend struct {
	Database *gorm.DB
}

// GetDB returns a handle to the db for CRUD operations
func (gb *GormBackend) GetDB() *gorm.DB {
	return gb.Database
}

// Base contains common columns for all tables.
type Base struct {
	ID        int `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time //`gorm:"index"`
}

// initDbBackend instantiates the DB specified by the config file. If this creates a new DB then
// additional steps are taken to create system accounts and groups.
func initDbBackend() {
	switch igor.Database.Adapter {
	case "sqlite":
		igor.IGormDb = NewSqliteGormBackend()
	default:
		igor.IGormDb = NewSqliteGormBackend()
	}

	db := igor.IGormDb.GetDB()

	// read existing clusters in db to populate acceptable reservation node range values
	cList, _ := dbReadClustersTx(nil)
	for _, c := range cList {

		ckeys := make([]int, 0, len(c.Hosts))
		for _, h := range c.Hosts {
			ckeys = append(ckeys, h.SequenceID)
		}
		sort.Ints(ckeys)

		r, _ := common.NewRange(c.Prefix, ckeys[0], ckeys[len(ckeys)-1])

		igor.ClusterRefs = append(igor.ClusterRefs, *r)
	}

	// Look for igor-admin in the database. If the account doesn't exist proceed with
	// first-time setup, otherwise we can stop here.
	_, status, uErr := getIgorAdminTx()
	if status == http.StatusOK {

		// If not doing first-time setup, check and update the MaxReserveTime value for default host policy if it's
		// been changed.
		if err := performDbTx(func(tx *gorm.DB) error {

			hpList, err := dbReadHostPolicies(map[string]interface{}{"name": DefaultPolicyName}, tx, &logger)
			if err != nil {
				return err
			}

			if len(hpList) == 0 {
				// someone dropped the host_policy table and we need to make a new default policy
				dur := time.Minute * time.Duration(igor.Scheduler.MaxReserveTime)
				allGroup, _, err := getAllGroup(tx)
				if err != nil || allGroup == nil {
					exitPrintFatal("Default HostPolicy missing, unable to recreate - no All Group found")
				}
				defaultPolicy := &HostPolicy{
					Name:         DefaultPolicyName,
					MaxResTime:   dur,
					AccessGroups: []Group{*allGroup},
					NotAvailable: ScheduleBlockArray{},
				}
				result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(defaultPolicy)
				if result.Error != nil {
					exitPrintFatal(fmt.Sprintf("database create default policy failed - %v", result.Error))
				}
				hpList, err = dbReadHostPolicies(map[string]interface{}{"name": DefaultPolicyName}, tx, &logger)
				if err != nil {
					return err
				}
			}

			configMaxResTime := time.Minute * time.Duration(igor.Scheduler.MaxReserveTime)

			if hpList[0].MaxResTime != configMaxResTime {
				if err = dbEditHostPolicy(hpList, map[string]interface{}{"maxResTime": configMaxResTime}, tx); err == nil {
					logger.Warn().Msgf("Updated maxResTime for default host policy to %d minutes", igor.Scheduler.MaxReserveTime)
				}
			}

			return err
		}); err != nil {
			exitPrintFatal(fmt.Sprintf("database error checking maxResTime update for default host policy - %v", err))
		}

		return

	} else if status >= http.StatusInternalServerError {
		exitPrintFatal(fmt.Sprintf("database error querying %s - %v", IgorAdmin, uErr))
	}

	passHash, _ := getPasswordHash(IgorAdmin)
	igorAdmin := &User{
		Name:     IgorAdmin,
		Email:    "",
		PassHash: passHash,
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(igorAdmin)
	if result.Error != nil {
		exitPrintFatal(fmt.Sprintf("database create %s - %v", IgorAdmin, result.Error))
	}

	// Make the general permissions for all users
	// allow anyone to view users, reservations, and hosts
	publicViewResources := PermUsers + PermSubpartToken +
		PermReservations + PermSubpartToken +
		PermHosts
	publicViewPermission := &Permission{
		Fact: NewPermissionString(publicViewResources, PermWildcardToken, PermViewAction),
	}

	publicCreateResources := PermGroups + PermSubpartToken +
		PermReservations + PermSubpartToken +
		PermDistros + PermSubpartToken +
		PermProfiles

	// allows anyone to create groups, reservations, distros and profiles
	publicCreatePermission := &Permission{
		Fact: NewPermissionString(publicCreateResources, PermWildcardToken, PermCreateAction),
	}

	// allows everyone to see members of the all group
	publicViewGroupAll := &Permission{
		Fact: NewPermissionString(PermGroups, GroupAll, PermViewAction),
	}

	// allows admins to see members of the admins group even when not elevated
	adminViewGroup := &Permission{
		Fact: NewPermissionString(PermGroups, GroupAdmins, PermViewAction),
	}

	var systemGroups = []Group{
		{
			Name:        GroupAll,
			Description: "igor users",
			Owner:       *igorAdmin,
			Members:     []User{*igorAdmin},
			Permissions: []Permission{*publicViewPermission, *publicCreatePermission, *publicViewGroupAll},
		},
		{
			Name:        GroupAdmins,
			Description: "igor administrators",
			Owner:       *igorAdmin,
			Members:     []User{*igorAdmin},
			Permissions: []Permission{*adminViewGroup},
		},
		{
			Name:          GroupUserPrefix + IgorAdmin,
			Description:   IgorAdmin + " private group",
			Owner:         *igorAdmin,
			Members:       []User{*igorAdmin},
			IsUserPrivate: true,
		},
	}
	result = db.Clauses(clause.OnConflict{DoNothing: true}).Create(systemGroups)
	if result.Error != nil {
		exitPrintFatal(fmt.Sprintf("database create system groups - %v", result.Error))
	}

	dur := time.Minute * time.Duration(igor.Scheduler.MaxReserveTime)

	defaultPolicy := &HostPolicy{
		Name:         DefaultPolicyName,
		MaxResTime:   dur,
		AccessGroups: []Group{systemGroups[0]},
		NotAvailable: ScheduleBlockArray{},
	}
	result = db.Clauses(clause.OnConflict{DoNothing: true}).Create(defaultPolicy)
	if result.Error != nil {
		exitPrintFatal(fmt.Sprintf("database create default policy - %v", result.Error))
	}

}

// performDbTx gets the backend database ref then calls the passed in method txFn that is
// expected for a GORM transaction, returning any errors.
func performDbTx(txFn func(tx *gorm.DB) error) error {
	db := igor.IGormDb.GetDB()
	return db.Transaction(txFn)
}
