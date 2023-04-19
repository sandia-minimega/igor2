// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"time"

	zl "github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doDeleteReservation deletes a reservation from the DB. It also removes the permissions for the reservation and the
// hosts it runs on (if the reservation was active). It ends by updating any node that was part of the reservation with
// a pending change to its access group (HostFuture).
func doDeleteReservation(resName string, r *http.Request) (status int, err error) {

	clog := hlog.FromRequest(r)
	actionUser := getUserFromContext(r)
	isElevated := userElevated(actionUser.Name)
	status = http.StatusInternalServerError // default status, overridden at end if no errors
	var res *Reservation
	var resClone *Reservation

	clusters, cErr := dbReadClustersTx(nil)
	if cErr != nil {
		return status, cErr
	}

	rList, grStatus, grErr := doReadReservations(map[string]interface{}{"name": resName}, map[string]time.Time{})
	if grErr != nil {
		status = grStatus
		return status, grErr
	}
	res = &rList[0]
	resClone = res.DeepCopy()

	// is this reservation running now or is it in the future?
	activeRes := res.Start.Before(time.Now())

	if err = performDbTx(func(tx *gorm.DB) error {
		status, err = doDeleteRes(res, tx, activeRes, clog)
		return err
	}); err == nil {
		status = http.StatusOK

		if hErr := resClone.HistCallback(resClone, HrDeleted); hErr != nil {
			clog.Error().Msgf("failed to record reservation '%s' delete to history", res.Name)
		}

		// Only send an email if the premature deletion was done by someone other than the owner
		if actionUser.Name != resClone.Owner.Name {
			if delEvent := makeResEditNotifyEvent(EmailResDelete, resClone, clusters[0].Name, actionUser, isElevated, ""); delEvent != nil {
				resNotifyChan <- *delEvent
			}
		}

		// power off the nodes and uninstall this res if it was active
		if activeRes {

			if err = uninstallRes(resClone); err != nil {
				status = http.StatusInternalServerError
				return
			}

			err = powerOffResNodes(resClone)
		}
	}

	return
}

// doDeleteRes deletes a reservation from the DB. It also removes the permissions for the reservation and the
// hosts it runs on (if the reservation was active). It ends by updating any node that was part of the reservation with
// a pending change to its access group (HostFuture).
func doDeleteRes(res *Reservation, tx *gorm.DB, activeRes bool, clog *zl.Logger) (status int, err error) {

	var hostList []Host

	// Look up all objects this reservation is part of and take action that may not allow delete to happen.
	// Return a 409-Conflict if the reservation cannot be deleted and include reason why
	//
	// If the server started and a reserved host is in an error state, can't change status to 'available'

	var perms []Permission
	perms, err = dbGetResourceOwnerPermissions(PermReservations, res.Name, &res.Owner, tx)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// if the profile group is the owner's private group all perms were picked up in previous step
	if res.Group.Name != (GroupUserPrefix + res.Owner.Name) {
		perms2, err2 := dbGetResourceGroupPermissions(PermReservations, res.Name, &res.Group, tx)
		if err2 != nil {
			return http.StatusInternalServerError, err2
		}
		perms = append(perms, perms2...)
	}

	// perform specific tasks if reservation is live (within start/end time)
	if activeRes {
		powerPerms, ppErr := dbGetHostPowerPermissions(&res.Group, res.Hosts, tx)
		if ppErr != nil {
			return http.StatusInternalServerError, ppErr
		}
		perms = append(perms, powerPerms...)

		// change the reservation's hosts out of 'reserved' state
		clog.Debug().Msgf("changing reservation %v's hosts out of 'reserved' state", res.Name)
		err = dbEditHosts(res.Hosts, map[string]interface{}{"State": HostAvailable}, tx)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	// grab a copy since the del op will get rid of the
	// res.Hosts one
	hostList = make([]Host, len(res.Hosts))
	for i, h := range res.Hosts {
		hostList[i] = h
	}

	if err = dbDeleteReservation(res, perms, activeRes, tx); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func uninstallRes(res *Reservation) (err error) {
	err = nil
	// skip if not using vlan
	if igor.Vlan.Network != "" {
		// clean up the network config
		if ncErr := networkClear(res.Hosts); ncErr != nil {
			err = fmt.Errorf("error clearing network isolation: %v", ncErr)
		}
	}

	// remove pxeboot configs for reservation hosts
	if uErr := igor.IResInstaller.Uninstall(res); err != nil {
		if err == nil {
			err = uErr
		} else {
			err = fmt.Errorf("%v\n%v", err, uErr)
		}
	}

	// power off the nodes of this reservation
	if pErr := powerOffResNodes(res); err != nil {
		if err == nil {
			err = pErr
		} else {
			err = fmt.Errorf("%v\n%v", err, pErr)
		}
	}

	// Put reservation nodes into maintenance mode if a Maintenance period has been specified
	if igor.Config.Maintenance.HostMaintenanceDuration > 0 {
		logger.Debug().Msgf("sending nodes for reservation %v into maintenance mode", res.Name)
		resetEnd := res.ResetEnd
		now := time.Now()
		// if the reservation is ending early, adjust the reset/maintenance time
		if now.Before(res.End) {
			// respect the maintenance padding at the time of res creation/extension
			delta := resetEnd.Sub(res.End)
			resetEnd = now.Add(delta)
		}
		// create a new MaintenanceRes from res
		maintenanceRes := &MaintenanceRes{
			ReservationName:    res.Name,
			MaintenanceEndTime: resetEnd,
			Hosts:              res.Hosts}
		err := dbCreateMaintenanceRes(maintenanceRes)
		if err != nil {
			logger.Error().Msgf("warning - errors detected when creating maintenance reservation %v: %v", res.Name, err)
		} else {
			// begin maintenance immediately
			startMaintenance(maintenanceRes)
		}
	}

	return err
}
