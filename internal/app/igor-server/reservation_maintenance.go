// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"time"

	"gorm.io/gorm"
)

type MaintenanceRes struct {
	Base
	ReservationName    string
	MaintenanceEndTime time.Time
	Hosts              []Host `gorm:"many2many:maintenanceres_hosts;"`
}

// dbCreateReservation puts a new reservation into the database.
func dbCreateMaintenanceRes(mRes *MaintenanceRes) (err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		result := tx.Create(&mRes)
		return result.Error
	})
	return err
}

// dbGetMaintenanceRes finds all maintenance Reservations within an existing transaction.
func dbGetMaintenanceRes() (resList []MaintenanceRes, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		result := tx.Preload("Hosts").Find(&resList)
		return result.Error
	})
	return resList, err
}

// dbUpdateReservation sets Started to True.
// func dbUpdateMaintenanceRes(mRes *MaintenanceRes) (err error) {
// 	err = performDbTx(func(tx *gorm.DB) error {
// 		result := tx.Model(&mRes).Update("started", true)
// 		return result.Error
// 	})
// 	return err
// }

// dbDeleteMaintenanceRes deletes the given MaintenanceRes from the DB
func dbDeleteMaintenanceRes(mRes *MaintenanceRes) (err error) {
	// delete the Maintenance reservation
	err = performDbTx(func(tx *gorm.DB) error {
		// delete the associations with the hosts table
		if clErr := tx.Model(&mRes).Association("Hosts").Clear(); clErr != nil {
			return clErr
		}
		result := tx.Delete(&mRes)
		return result.Error
	})
	return err
}
