// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"

	"igor2/internal/pkg/common"
)

func doUpdateReservation(resName string, editParams map[string]interface{}, r *http.Request) (status int, err error) {

	status = http.StatusInternalServerError // default status, overridden at end if no errors
	clog := hlog.FromRequest(r)
	var res *Reservation
	actionUser := getUserFromContext(r)
	clog.Debug().Msgf("update reservation: '%s' by user %s with params %+v", resName, actionUser.Name, editParams)
	isElevated := userElevated(actionUser.Name)
	var extended, renamed, dropped, isNewOwner, isNewGroup bool
	var clusterName, oldName, newOwnerName string
	var oldOwner User
	var droppedHosts, addHosts []Host

	if err = performDbTx(func(tx *gorm.DB) error {

		clusters, cErr := dbReadClustersTx(nil)
		if cErr != nil {
			return cErr
		}
		clusterName = clusters[0].Name

		rList, ruStatus, ruErr := getReservations([]string{resName}, tx)
		if ruErr != nil {
			status = ruStatus
			return ruErr
		}

		res = &rList[0]
		oldName = res.Name
		oldOwner = res.Owner
		extendDur, doExtendS := editParams["extend"].(string)
		extendTime, doExtendF := editParams["extend"].(float64)
		dropList, doDrop := editParams["drop"].(string)
		addCount, doAddByVal := editParams["addNodeCount"].(float64)
		addList, doAddByList := editParams["addNodeList"].(string)
		_, doExtendMax := editParams["extendMax"]
		_, doDistro := editParams["distro"]
		_, doProfile := editParams["profile"]
		_, renamed = editParams["name"]
		newOwnerName, isNewOwner = editParams["owner"].(string)
		_, isNewGroup = editParams["group"]
		var changes map[string]interface{}
		var vErr error
		if doExtendF || doExtendS || doExtendMax {

			if igor.Scheduler.ExtendWithin < 0 {
				if !isElevated {
					status = http.StatusBadRequest
					return fmt.Errorf("extending a reservation has been disabled for normal users - talk to an igor admin if you wish to change your reservation end time")
				} else {
					clog.Info().Msgf("user '%s' is invoking admin privileges to extend reservation '%s' when extensions have been disabled", actionUser.Name, resName)
				}
			}

			extended = true
			if doExtendF {
				extendDur = time.Unix(int64(extendTime), 0).Format(common.DateTimeCompactFormat)
			}
			changes, status, vErr = parseExtend(res, extendDur, isElevated, r, tx)
		} else if isNewOwner && newOwnerName == IgorAdmin {
			status = http.StatusBadRequest
			clog.Warn().Msgf("'%s' unsuccessully attempted to change reservation owner of '%s' to igor-admin", actionUser.Name, resName)
			return fmt.Errorf("cannot change reservation '%s' owner to igor-admin", resName)
		} else if doDrop {
			changes, status, vErr = parseDrop(res, dropList, tx)
			if vErr == nil {
				dropped = true
				droppedHosts = changes["dropHosts"].([]Host)
			}
		} else if doAddByList || doAddByVal {
			changes = map[string]interface{}{}
			var hostNames []string
			dummyRes := res.DeepCopy()

			if doAddByList {
				// get the host names so we can look at the schedule
				hostNames = igor.splitRange(addList)
				addHosts, status, err = getHosts(hostNames, true, tx)
				if err != nil {
					return err
				}
				dummyRes.Hosts = addHosts
				// verify named hosts are available
				if status, err = scheduleHostsByName(dummyRes, tx, clog); err != nil {
					return err
				}
			} else {
				if addCount < 1 {
					status = http.StatusBadRequest
					return fmt.Errorf("must include at least one host if adding to reservation")
				}
				dummyRes.Hosts = make([]Host, int(addCount))
				if addHosts, status, err = scheduleHostsByAvailability(dummyRes, tx, clog); err != nil {
					return err
				}
			}
			// Check against allowed host max limit when not an elevated admin
			totalHosts := len(addHosts) + len(res.Hosts)
			if !isElevated && igor.Scheduler.NodeReserveLimit > 0 && totalHosts > igor.Scheduler.NodeReserveLimit {
				err = fmt.Errorf("host reserve limit exceeded if new hosts are added, reservation cannot have more than %v hosts", igor.Scheduler.NodeReserveLimit)
				clog.Warn().Msgf("%v", err)
				status = http.StatusForbidden
				return err
			}
			changes["addHosts"] = addHosts

		} else if doDistro || doProfile {
			changes, status, vErr = parseImageEdits(res, editParams, tx)
		} else {
			changes, status, vErr = parseResEditParams(res, editParams, tx)
		}
		if vErr != nil {
			return vErr
		}

		if (len(addHosts) > 0) && (res.Installed || (res.Start.Before(time.Now()) && time.Now().Before(res.End))) {
			// if the reservation is active, delete the power perms so we can rebuild them below with the new hosts included
			old_power_perms, err := dbGetHostPowerPermissions(&res.Group, res.Hosts, tx)
			if err != nil {
				return err
			}
			// delete the permissions for this reservation
			result := tx.Delete(old_power_perms)
			if result.Error != nil {
				return result.Error
			}
		}

		return dbEditReservation(res, changes, tx)

	}); err != nil {
		return
	}

	status = http.StatusOK

	if dropped {
		if vlanErr := networkClear(droppedHosts); vlanErr != nil {
			clog.Error().Msgf("vlan error on res node drop - %v", vlanErr)
		}
		if _, powerErr := doPowerHosts(PowerOff, hostNamesOfHosts(droppedHosts), clog); powerErr != nil {
			clog.Error().Msgf("problem powering off dropped hosts for reservation '%s': %v", resName, powerErr)
		}

		if igor.Config.Maintenance.HostMaintenanceDuration > 0 {
			logger.Debug().Msgf("putting dropped node(s) for reservation '%s' into maintenance mode", resName)

			// prep for saving the current state so it can be restored after maintenance mode is finished
			for _, h := range droppedHosts {
				h.RestoreState = HostAvailable // a dropped host will always return to available
			}

			now := time.Now()
			maintenanceDelta := time.Duration(float64(time.Minute) * float64(igor.Config.Maintenance.HostMaintenanceDuration))
			maintenanceEnd := now.Add(maintenanceDelta)
			// create a new MaintenanceRes from res
			maintenanceResDrop := &MaintenanceRes{
				ReservationName:    res.Name + "-nodeDrop",
				MaintenanceEndTime: maintenanceEnd,
				Hosts:              droppedHosts}
			cmErr := dbCreateMaintenanceRes(maintenanceResDrop)
			if cmErr != nil {
				logger.Error().Msgf("warning - errors detected when creating dropped node maintenance reservation %s: %v", res.Name, cmErr)
			} else {
				// begin maintenance immediately
				_ = startMaintenance(maintenanceResDrop)
			}
		}
	}

	// Install these hosts if the reservation is active
	if (len(addHosts) > 0) && (res.Installed || (res.Start.Before(time.Now()) && time.Now().Before(res.End))) {
		if err = performDbTx(func(tx *gorm.DB) error {
			// var result *gorm.DB
			err = dbEditHosts(addHosts, map[string]interface{}{"State": HostReserved}, tx)
			if err != nil {
				return err
			}

			// create and add power perms
			powerPerm, permErr := NewPermission(makeNodePowerPerm(res.Hosts))
			if permErr != nil {
				return permErr
			}
			if apErr := dbAppendPermissions(&res.Group, []Permission{*powerPerm}, tx); apErr != nil {
				return apErr
			}

			// skip if not using vlan
			if igor.Vlan.Network != "" {
				// update network config
				if nsErr := networkSet(addHosts, res.Vlan); nsErr != nil {
					return fmt.Errorf("error setting network isolation: %v", nsErr)
				}
			}
			dummyRes := res.DeepCopy()
			dummyRes.Hosts = addHosts
			// install the reservation's profile to its hosts
			logger.Debug().Msgf("installing PXE files to added Hosts for reservation %s", dummyRes.Name)
			if irErr := igor.IResInstaller.Install(dummyRes); irErr != nil {
				// update the reservation with the error message
				if irErr = dbEditReservation(res, map[string]interface{}{"install_error": irErr.Error()}, tx); irErr != nil {
					return irErr
				}
				return irErr
			}

			if res.CycleOnStart {
				logger.Debug().Msgf("power cycling hosts for reservation '%s'", res.Name)
				if _, powerErr := doPowerHosts(PowerCycle, hostNamesOfHosts(addHosts), &logger); powerErr != nil {
					// don't return this error we still want to mark it installed
					logger.Error().Msgf("problem powering cycling hosts for the added hosts for reservation '%s': %v", res.Name, powerErr)
				}
			} else {
				logger.Warn().Msgf("The added hosts for reservation '%s' were not powered cycled at start", res.Name)
			}
			return nil
		}); err != nil {
			return
		}
	}
	rList, _ := dbReadReservationsTx(map[string]interface{}{"ID": res.ID}, nil)
	res = &rList[0]

	editKeys := make([]string, 0, len(editParams))
	for k := range editParams {
		editKeys = append(editKeys, k)
	}
	sort.Strings(editKeys)

	if hErr := res.HistCallback(res, HrUpdated+":"+strings.Join(editKeys, ",")); hErr != nil {
		logger.Error().Msgf("failed to record reservation '%s' update to history", res.Name)
	}

	var editEvents []*ResNotifyEvent

	if dropped && actionUser.Name != res.Owner.Name {
		dropList := common.UnsplitList(hostNamesOfHosts(droppedHosts))
		if resEditEvent := makeResEditNotifyEvent(EmailResDrop, res, clusterName, actionUser, isElevated, dropList); resEditEvent != nil {
			editEvents = append(editEvents, resEditEvent)
		}
	}

	if extended && actionUser.Name != res.Owner.Name {
		if resEditEvent := makeResEditNotifyEvent(EmailResExtend, res, clusterName, actionUser, isElevated, ""); resEditEvent != nil {
			editEvents = append(editEvents, resEditEvent)
		}
	}

	if renamed {
		if resEditEvent := makeResEditNotifyEvent(EmailResRename, res, clusterName, actionUser, isElevated, oldName); resEditEvent != nil {
			editEvents = append(editEvents, resEditEvent)
		}
	}

	if isNewOwner {
		if resEditEvent := makeResEditNotifyEvent(EmailResNewOwner, res, clusterName, &oldOwner, false, ""); resEditEvent != nil {
			editEvents = append(editEvents, resEditEvent)
		}
	}

	if isNewGroup && !strings.HasPrefix(res.Group.Name, GroupUserPrefix) {
		if resEditEvent := makeResEditNotifyEvent(EmailResNewGroup, res, clusterName, actionUser, isElevated, ""); resEditEvent != nil {
			editEvents = append(editEvents, resEditEvent)
		}
	}

	if len(editEvents) > 0 {
		for _, event := range editEvents {
			resNotifyChan <- *event
		}
	}

	return
}

func parseDrop(res *Reservation, dropList string, tx *gorm.DB) (map[string]interface{}, int, error) {

	changes := map[string]interface{}{}

	dropHostList := igor.splitRange(dropList)

	dropHosts := make([]Host, 0, len(dropHostList))

	for _, dh := range dropHostList {
		found := false
		for _, rh := range res.Hosts {
			if dh == rh.Name {
				dropHosts = append(dropHosts, rh)
				found = true
				break
			}
		}
		if !found {
			return nil, http.StatusNotFound, fmt.Errorf("%s was not a part of reservation '%s'", dh, res.Name)
		}
	}

	if len(dropHosts) == len(res.Hosts) {
		return nil, http.StatusBadRequest, fmt.Errorf("dropping all nodes from reservation not allowed - use delete instead")
	}

	changes["dropHosts"] = dropHosts

	now := time.Now()

	if res.Installed || (res.Start.Before(now) && now.Before(res.End)) {
		changes["resIsNow"] = true
		if powerPerms, err := dbGetHostPowerPermissions(&res.Group, res.Hosts, tx); err != nil {
			return nil, http.StatusInternalServerError, err
		} else {
			powerPerm := powerPerms[0]
			keepHosts := make([]Host, 0, len(res.Hosts)-len(dropHosts))
			for _, h := range res.Hosts {
				isKeep := true
				for _, dh := range dropHosts {
					if dh.ID == h.ID {
						isKeep = false
						break
					}
				}
				if isKeep {
					keepHosts = append(keepHosts, h)
				}
			}
			pUpdate, _ := NewPermission(makeNodePowerPerm(keepHosts))
			pUpdate.ID = powerPerm.ID
			pUpdate.GroupID = powerPerm.GroupID
			changes["pUpdate"] = *pUpdate
		}
	}

	return changes, http.StatusOK, nil
}

// parseExtend checks that the 'extend' parameter has correct syntax and the modified end time
// it creates doesn't collide with existing reservations and/or host policies.
func parseExtend(res *Reservation, extendTime string, isActionUserElevated bool, r *http.Request, tx *gorm.DB) (map[string]interface{}, int, error) {

	clog := hlog.FromRequest(r)

	if !isActionUserElevated {
		for _, h := range res.Hosts {
			if h.State == HostBlocked {
				return nil, http.StatusConflict,
					fmt.Errorf("cannot extend a reservation containing nodes with a blocked status -- contact cluster admin team")
			}
		}
	}

	hostNameList := namesOfHosts(res.Hosts)

	// get the smallest host policy max limit
	smallestMaxTime := time.Duration(math.MaxInt64)

	hostIDs, status, err := getHostIDsFromNames(hostNameList)
	if err != nil {
		return nil, status, err
	} else {
		if hpList, rhpErr := dbReadHostPolicies(map[string]interface{}{"hosts": hostIDs}, tx, clog); rhpErr != nil {
			return nil, http.StatusInternalServerError, rhpErr
		} else {
			for _, hp := range hpList {
				if hp.MaxResTime < smallestMaxTime {
					smallestMaxTime = hp.MaxResTime
				}
			}
		}
	}

	now := time.Now()
	var extendDur time.Duration

	if extendTime == "" {
		// extend by maximum allowable
		extendDur = (smallestMaxTime - res.Remaining(now)).Truncate(time.Minute)
	} else {
		// extend by provided parameter, either a duration or a datetime stamp
		if extendDur, err = common.ParseDuration(extendTime); err != nil {
			if extendDts, pErr := common.ParseTimeFormat(extendTime); pErr != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("%v; and, %v", err, pErr)
			} else {
				if !extendDts.After(res.End) {
					return nil, http.StatusBadRequest, fmt.Errorf("extend datetime '%s' is earlier than original '%s'",
						extendDts.Format(common.DateTimeCompactFormat), res.End.Format(common.DateTimeCompactFormat))
				}
				extendDur = extendDts.Sub(res.End).Truncate(time.Minute)
			}
		}
	}

	newEndTime := res.End.Add(extendDur).Round(time.Minute)
	// determine new reset/maintenance end time from newEndTime
	resetEnd := determineNodeResetTime(newEndTime)

	// if this is not an elevated admin check for time limits, otherwise pass-through
	if !isActionUserElevated {
		// Make sure the reservation doesn't exceed max allowable time for the given number of nodes
		if err = checkTimeLimit(len(res.Hosts), smallestMaxTime, res.Remaining(now)+extendDur); err != nil {
			return nil, http.StatusBadRequest, err
		}

		// Make sure that the user is extending a reservation that is near its completion based on the ExtendWithin config.
		if igor.Scheduler.ExtendWithin > 0 {
			remaining := time.Until(res.End)
			if int(remaining.Minutes()) > igor.Scheduler.ExtendWithin {
				ewDur := common.FormatDuration(time.Minute*time.Duration(igor.Scheduler.ExtendWithin), false)
				return nil, http.StatusBadRequest, fmt.Errorf("reservations can only be extended if they are within %v of ending", ewDur)
			}
		}
	}

	if err = checkScheduleLimit(newEndTime, isActionUserElevated); err != nil {
		return nil, http.StatusBadRequest, err
	}

	// verify extension doesn't conflict with current host policies
	groupAccessList := groupNamesOfGroups(res.Owner.Groups)
	checkStart := res.Start
	if res.Installed {
		checkStart = now
	}
	if hpStatus, hpErr := dbCheckHostPolicyConflicts(hostNameList, groupAccessList, userElevated(res.Owner.Name), checkStart, res.End, newEndTime, clog); hpErr != nil {
		return nil, hpStatus, hpErr
	}

	// verify extension (plus maintenance, if any) doesn't conflict with existing future reservations utilizing the same hosts
	resList, rrErr := dbReadReservations(map[string]interface{}{"hosts": hostIDs}, nil, tx)
	if rrErr != nil {
		return nil, http.StatusInternalServerError, rrErr
	}

	for _, otherRes := range resList {
		if res.Name != otherRes.Name {
			if otherRes.Start.Before(resetEnd) {
				return nil, http.StatusConflict, fmt.Errorf("cannot extend reservation; one or more hosts are reserved prior to the proposed new end time")
			}
		}
	}

	changes := map[string]interface{}{}
	changes["End"] = newEndTime
	changes["ResetEnd"] = resetEnd
	changes["ExtendCount"] = res.ExtendCount + 1

	if !*igor.Email.ResNotifyOn || newEndTime.Sub(now) < ResNotifyTimes[0] {
		changes["NextNotify"] = time.Duration(0)
	} else {
		for i := len(ResNotifyTimes) - 1; i >= 0; i-- {
			if newEndTime.Sub(now) >= ResNotifyTimes[i] {
				changes["NextNotify"] = ResNotifyTimes[i]
				break
			}
		}
	}

	return changes, http.StatusOK, nil
}

// parseImageEdits ensures that the reservation owner has access to the new distro and/or profile
// specified in the change.
func parseImageEdits(res *Reservation, editParams map[string]interface{}, tx *gorm.DB) (map[string]interface{}, int, error) {

	var newDistro *Distro
	var newProfile *Profile
	changes := map[string]interface{}{}

	if newProfileName, ok := editParams["profile"].(string); ok {
		// make sure new profile exists
		newProfileName = strings.TrimSpace(newProfileName)
		if pList, err := dbReadProfiles(map[string]interface{}{"name": newProfileName, "owner_id": res.Owner.ID}, tx); err != nil {
			return changes, http.StatusInternalServerError, err
		} else if len(pList) == 0 {
			return changes, http.StatusConflict, fmt.Errorf("no profiles returned for user %v with name %v", res.Owner.Name, newProfileName)
		} else {
			newProfile = &pList[0]
			// make sure the distro of this profile is still accessible to the user
			if dList, status, err := getDistros([]string{newProfile.Distro.Name}, tx); err != nil {
				return changes, status, err
			} else if len(dList) == 0 {
				return changes, http.StatusConflict, fmt.Errorf("no distro returned with name %v", newProfile.Distro.Name)
			} else {
				newDistro = &dList[0]
				if !res.Owner.isMemberOfAnyGroup(newDistro.Groups) {
					return nil, http.StatusForbidden, fmt.Errorf("%s does not currently have access to distro '%s' in profile '%s'", res.Owner.Name, newDistro.Name, newProfileName)
				}
			}
			changes["profile"] = newProfile
		}

	} else if newDistroName, ok := editParams["distro"].(string); ok {
		// make sure the distro exists and user can access it
		newDistroName = strings.TrimSpace(newDistroName)
		if dList, status, err := getDistros([]string{newDistroName}, tx); err != nil {
			return changes, status, err
		} else if len(dList) == 0 {
			return changes, http.StatusConflict, fmt.Errorf("no distro returned with name %v", newDistroName)
		} else {
			newDistro = &dList[0]
			if !res.Owner.isMemberOfAnyGroup(newDistro.Groups) {
				return nil, http.StatusForbidden, fmt.Errorf("%s does not have access to distro '%s'", res.Owner.Name, newDistro.Name)
			}
			changes["profile"] = &Profile{
				Name:        generateDefaultProfileName(&res.Owner),
				Owner:       res.Owner,
				Distro:      *newDistro,
				IsDefault:   true,
				Description: "Default profile for distro " + newDistro.Name + " for reservation " + res.Name,
			}
			changes["create_new_profile"] = true
		}
	}

	return changes, http.StatusOK, nil
}

func parseResEditParams(res *Reservation, editParams map[string]interface{}, tx *gorm.DB) (map[string]interface{}, int, error) {

	var newOwner *User
	var err error
	changes := map[string]interface{}{}

	// check if the reservation name is changing
	if name, ok := editParams["name"].(string); ok {
		changes["Name"] = name
	}

	// check if the description is changing
	if desc, ok := editParams["description"].(string); ok {
		changes["Description"] = desc
	}

	// does user want to add kernel args to the temp profile?
	kernelArgs, kOk := editParams["kernelArgs"].(string)
	if kOk {
		if res.Profile.IsDefault {
			// ok to modify a temp profile
			changes["profile_kernel"] = kernelArgs
		} else {
			return changes, http.StatusBadRequest, fmt.Errorf("cannot modify permanent profile, edit the profile first")
		}
	}
	newOwnerName, ownOK := editParams["owner"].(string)
	groupName, grpOK := editParams["group"].(string)

	if !ownOK && !grpOK {
		return changes, http.StatusOK, nil
	}

	var pgChanges []Permission
	var distroGroups []Group
	distroName := res.Profile.Distro.Name

	// if we are changing owners or adding/changing to a non-pug group we'll need distro information
	if ownOK || (grpOK && groupName != GroupNoneAlias) {
		if dList, status, err := getDistros([]string{distroName}, tx); err != nil {
			return nil, status, err
		} else {
			distroGroups = dList[0].Groups
		}
	}

	// get the current power perms (will be empty if reservation hasn't started yet)
	powerPerms, ppErr := dbGetHostPowerPermissions(&res.Group, res.Hosts, tx)
	if ppErr != nil {
		return nil, http.StatusInternalServerError, ppErr
	}

	// check if the owner is changing
	if ownOK {
		// get the user object for the new owner
		uList, status, guErr := getUsers([]string{newOwnerName}, false, tx)
		if guErr != nil {
			return nil, status, guErr
		}
		newOwner = &uList[0]

		// make sure the new owner can use the reservation's distro
		if !newOwner.isMemberOfAnyGroup(distroGroups) && newOwner.Name != IgorAdmin {
			return nil, http.StatusForbidden, fmt.Errorf("%s does not have access to distro '%s'", newOwner.Name, distroName)
		} else {
			// duplicate the profile into a new default profile for the new owner
			changes["create_new_profile"] = true
			dupProfile := res.Profile.duplicate(newOwner)
			dupProfile.Name = generateDefaultProfileName(newOwner)
			dupProfile.IsDefault = true
			// Don't interact with the DB here, use the res update transaction to insert the new profile to the db
			changes["profile"] = dupProfile
		}
		changes["OwnerID"] = newOwner.ID

		// also make sure the new owner isn't restricted from any of the reservation's hosts' policies
		hostNames := namesOfHosts(res.Hosts)
		// get all host-policies associated with the given list of host names
		myHostPolicies, err := getHostPoliciesFromHostNames(hostNames)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		// make a list of the access groups that this new owner qualifies for
		var groupAccessList []string
		for _, uGroup := range newOwner.Groups {
			if !strings.HasPrefix(uGroup.Name, GroupUserPrefix) {
				groupAccessList = append(groupAccessList, uGroup.Name)
			}
		}
		// determine if any policies do not contain at least one group from groupAccessList
		if membership, policy := dbCheckHostPolicyGroupConflicts(myHostPolicies, groupAccessList); !membership {
			// get the intersection of affected policy hosts and requested hosts
			offendingHosts := getHostIntersection(hostNames, policy.Hosts)
			return nil, http.StatusConflict, &HostPolicyConflictError{"no group available that matches node restriction", true, false, false, time.Time{}, time.Time{}, offendingHosts}
		}

		// if the reservation group is not going to change (and not a pug), make sure the new owner is also a member
		if !grpOK && !res.Group.IsUserPrivate {
			if userElevated(res.Owner.Name) && newOwner.Name == IgorAdmin {
				// fall through
			} else if !groupSliceContains(newOwner.Groups, res.Group.Name) && newOwner.Name != IgorAdmin {
				return nil, http.StatusConflict, fmt.Errorf("new owner is not a member of current reservation group %v", res.Group.Name)
			}
		}

		// get the new owner pug id
		newPugID, ggErr := newOwner.getPugID()
		if ggErr != nil {
			return nil, http.StatusInternalServerError, ggErr
		}
		changes["p-owner-gid"] = newPugID

		// get current owner permissions to transfer to new owner
		poChanges, gpErr := dbGetResourceOwnerPermissions(PermReservations, res.Name, &res.Owner, tx)
		if gpErr != nil {
			return nil, http.StatusInternalServerError, gpErr
		}
		changes["owner-perms"] = poChanges

		// if group is being dropped OR no group change but current group is pug
		// prep the group permissions to change to the new owner
		if (grpOK && groupName == GroupNoneAlias) || (!grpOK && res.Group.IsUserPrivate) {
			// determine group permissions to transfer to new owner
			changes["GroupID"] = newPugID
			changes["p-gid"] = newPugID

			pgChanges, gpErr = dbGetResourceGroupPermissions(PermReservations, res.Name, &res.Group, tx)
			if gpErr != nil {
				return nil, http.StatusInternalServerError, gpErr
			}

			// if there are already power permissions prep to change to the new owner
			if len(powerPerms) > 0 {
				// if they do, add them to the change list
				pgChanges = append(pgChanges, powerPerms...)
			}
			changes["group-perms"] = pgChanges
		}
	}

	if grpOK {
		// if dropping the group...
		if groupName == GroupNoneAlias {
			if !ownOK {
				newPugID, ggErr := res.Owner.getPugID()
				if ggErr != nil {
					return nil, http.StatusInternalServerError, ggErr
				}
				changes["GroupID"] = newPugID
				changes["p-gid"] = newPugID

				pgChanges, err = dbGetResourceGroupPermissions(PermReservations, res.Name, &res.Group, tx)
				if err != nil {
					return nil, http.StatusInternalServerError, err
				}

				// if there are already power permissions prep to change to the new group
				if len(powerPerms) > 0 {
					// if they do, add them to the change list
					pgChanges = append(pgChanges, powerPerms...)
				}
			}
		} else {

			gList, status, err := getGroupsTx([]string{groupName}, true)
			if err != nil {
				return nil, status, err
			}
			newGroup := &gList[0]

			if ownOK && !newOwner.isMemberOfGroup(newGroup) {
				return nil, http.StatusForbidden, fmt.Errorf("user '%s' is not a member of group '%s'", newOwner.Name, groupName)
			}

			if !ownOK && !res.Owner.isMemberOfGroup(newGroup) {
				return nil, http.StatusForbidden, fmt.Errorf("current owner '%s' is not a member of group '%s'", res.Owner.Name, groupName)
			}

			changes["GroupID"] = newGroup.ID
			changes["p-gid"] = newGroup.ID

			// This is a little faster than making multiple calls to dbGetResourceGroupPermissions
			pList := makeResGroupPermStrings(res)
			pgChanges, err = dbGetPermissions(map[string]interface{}{"fact": pList}, tx)
			if err != nil {
				return nil, http.StatusInternalServerError, err
			}

			// if there are already power permissions prep to change to the new group
			if len(powerPerms) > 0 {
				// if they do, add them to the change list
				pgChanges = append(pgChanges, powerPerms...)
			}
		}

		changes["group-perms"] = pgChanges
	}

	return changes, http.StatusOK, nil
}
