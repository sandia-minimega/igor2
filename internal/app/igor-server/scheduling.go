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

	zl "github.com/rs/zerolog"
	"gorm.io/gorm"

	"igor2/internal/pkg/common"
)

// scheduleHostsByName finds the first time the specified hosts are free for the requested duration.
func scheduleHostsByName(res *Reservation, tx *gorm.DB, clog *zl.Logger) (int, error) {

	// Search to see if there are any reservations on the given hosts that end
	// on or after the start time of the new reservation.
	hostNameList := namesOfHosts(res.Hosts)

	// make a list of the access groups that this user qualifies for
	groupAccessList := []string{}
	for _, uGroup := range res.Owner.Groups {
		if !strings.HasPrefix(uGroup.Name, GroupUserPrefix) {
			groupAccessList = append(groupAccessList, uGroup.Name)
		}
	}

	// check if all hosts are in an available state
	status, err := dbCheckHostAvailable(hostNameList, tx)
	if err != nil {
		return status, err
	}

	// check that no hosts have conflicts in their host policy
	isElevated := userElevated(res.Owner.Name)
	status, err = dbCheckHostPolicyConflicts(hostNameList, groupAccessList, isElevated, res.Start, res.End, res.End, clog)
	if err != nil {
		return status, err
	}

	// finally, make sure the hosts aren't already being used for the requested reservation times
	_, status, err = dbCheckResvConflicts(hostNameList, res.Start, res.End, tx)
	if err != nil {
		return status, err
	}

	return status, nil
}

// scheduleHostsByAvailability finds a suitable block of hosts that are free for the requested duration. If one
// contiguous block isn't available it will find the smallest number of contiguous blocks possible.
func scheduleHostsByAvailability(res *Reservation, tx *gorm.DB, clog *zl.Logger) ([]Host, int, error) {

	numHostsReq := len(res.Hosts) // number of hosts needed for res
	isElevated := userElevated(res.Owner.Name)

	groupAccessList := []string{GroupAll}
	if !strings.HasPrefix(res.Group.Name, GroupUserPrefix) {
		groupAccessList = append(groupAccessList, res.Group.Name)
	}

	validAccessHosts, status, err := dbGetAccessibleHosts(groupAccessList, isElevated, res.Start, res.End, numHostsReq, tx, clog)
	if err != nil {
		return nil, status, err
	}

	// get open slots for each set of hosts
	validOpenSlotMap := make(map[string][]ReservationTimeSlot)
	var hasRestrictedHosts bool
	totalHostAvail := 0
	// Calculate end time to use including any configured maintenence padding
	paddedEndTime := determineNodeResetTime(res.End)
	paddedDur := paddedEndTime.Sub(res.Start)

	for ahKey, ahList := range validAccessHosts {
		ahNames := namesOfHosts(ahList)
		if ahKey != DefaultPolicyName {
			hasRestrictedHosts = true
		}
		openSlots, osStatus, osErr := dbFindOpenSlots(ahNames, res.Start, paddedDur, getScheduleEnd(isElevated), numHostsReq, tx)
		if osErr != nil {
			return nil, osStatus, osErr
		}

		tempValid := openSlots[:0]
		for _, s := range openSlots {
			// if the slot doesn't start after the res start time and res duration can fit inside the
			// slot, these are valid. anything else is dropped because it won't work
			if !s.AvailSlotBegin.After(res.Start) && paddedDur <= s.AvailSlotEnd.Sub(res.Start) {
				tempValid = append(tempValid, s)
			}
		}

		totalHostAvail += len(tempValid)
		validOpenSlotMap[ahKey] = tempValid
	}

	// Now we have all the available nodes that can be scheduled during this reservation's requested time slot
	if totalHostAvail < numHostsReq {
		return nil, http.StatusConflict,
			fmt.Errorf("%v hosts cannot be found with enough time available to service this request", numHostsReq)
	}

	hostNameList := findBestSolution(validOpenSlotMap, hasRestrictedHosts, numHostsReq)

	// now go get those hosts!
	queryParams := map[string]interface{}{"name": hostNameList}
	hostResList, rhErr := dbReadHosts(queryParams, tx)
	if rhErr != nil {
		return nil, http.StatusInternalServerError, rhErr
	}

	return hostResList, http.StatusOK, nil
}

// findBestSolution picks the smallest number of contiguous segments it needs to make the reservation. If the reservation
// includes a group that is part of a node restriction policy, it will attempt to prioritize use of the policy's nodes first
// before grabbing nodes from the general open pool of nodes. It returns a list of hostnames included in the segment(s).
func findBestSolution(validOpenSlotMap map[string][]ReservationTimeSlot, withRestrictedHosts bool, numHostsReq int) []string {

	hostNameList := make([]string, numHostsReq)
	validOpenSlots := make([]ReservationTimeSlot, 0)

	if withRestrictedHosts {

		closestSize := math.MaxInt32
		var closestPolicy string
		for ahKey, ahList := range validOpenSlotMap {
			if ahKey == DefaultPolicyName {
				continue
			}
			closeness := numHostsReq - len(ahList)
			// we want to favor exact matches, then slot lists with more capacity than needed over
			// slot lists that would be used up and still need nodes from default
			if closeness == 0 {
				// we got an exact match
				closestPolicy = ahKey
				break
			} else if closeness < closestSize && closestSize > 0 {
				// keep using larger arrays of open slots until they exceed numHostsReq
				closestSize = closeness
				closestPolicy = ahKey
			} else if closeness > closestSize && closeness < 0 {
				// but once you are using excess capacity slot lists keep them as small as possible
				closestSize = closeness
				closestPolicy = ahKey
			}
		}

		if len(validOpenSlotMap[closestPolicy]) <= numHostsReq {

			for i, s := range validOpenSlotMap[closestPolicy] {
				hostNameList[i] = s.Hostname
			}
			if len(validOpenSlotMap[closestPolicy]) == numHostsReq {
				// we can cover the res with the closest policy nodes -- done!
				return hostNameList
			}
			// we need more nodes - use default policy nodes
			validOpenSlots = validOpenSlotMap[DefaultPolicyName]
		} else {
			// otherwise the closest policy match has more then enough valid slots to
			// make the reservation, so pick from those
			validOpenSlots = validOpenSlotMap[closestPolicy]
		}
	} else {
		// we are only using default policy nodes to make the reservation
		validOpenSlots = validOpenSlotMap[DefaultPolicyName]
	}

	// sort by sequence number to bin into contiguous blocks
	sort.Slice(validOpenSlots, func(i, j int) bool {
		return validOpenSlots[i].Hostnum < validOpenSlots[j].Hostnum
	})

	// Group all contiguous blocks into separate lists. There is a maximum of len(validOpenSlots) lists
	// when no contiguous blocks exist.
	cbList := make([][]ReservationTimeSlot, len(validOpenSlots))
	cbList[0] = append(cbList[0], validOpenSlots[0])
	if len(validOpenSlots) > 1 {
		var offset = 1
		for i := 1; i < len(validOpenSlots); i++ {
			if validOpenSlots[i].Hostnum == validOpenSlots[i-1].Hostnum+1 {
				cbList[i-offset] = append(cbList[i-offset], validOpenSlots[i])
				offset++
			} else {
				cbList[i] = append(cbList[i], validOpenSlots[i])
				offset = 1
			}
		}
	}

	// Now sort the lists by size from largest to smallest
	sort.Slice(cbList, func(i, j int) bool {
		return len(cbList[i]) > len(cbList[j])
	})

	// Assign nodes using the smallest number of contiguous blocks possible. Blocks of exact size needed will be
	// picked first, followed by taking required nodes from a single larger block, and finally using smaller
	// blocks of decreasing size.

	// subtract the number of hosts already assigned restricted nodes
	var stillNeeded = numHostsReq
	for _, hn := range hostNameList {
		if hn != "" {
			stillNeeded--
		}
	}
	var filled = numHostsReq - stillNeeded
	var assigned []int
	for stillNeeded > 0 {
		var moreCapacity []int
		var lessCapacity []int
	InnerLoop:
		for i := 0; i < len(cbList); i++ {
			if len(assigned) > 0 {
				for _, a := range assigned {
					if i == a {
						continue InnerLoop
					}
				}
			}
			if len(cbList[i]) == stillNeeded {
				assigned = append(assigned, i)
				stillNeeded = 0
				break InnerLoop
			} else if len(cbList[i]) > stillNeeded {
				moreCapacity = append(moreCapacity, i)
			} else if len(cbList[i]) < stillNeeded && len(cbList[i]) > 0 {
				lessCapacity = append(lessCapacity, i)
			} else {
				continue
			}
		}

		if stillNeeded > 0 {
			if len(moreCapacity) > 0 {
				smc := moreCapacity[len(moreCapacity)-1]
				assigned = append(assigned, smc)
				stillNeeded -= len(cbList[smc])
			} else {
				llc := lessCapacity[0]
				assigned = append(assigned, llc)
				stillNeeded -= len(cbList[llc])
			}
		}
	}

	k := filled
CbLoop:
	for _, i := range assigned {
		for j := 0; j < len(cbList[i]); j++ {
			hostNameList[k] = cbList[i][j].Hostname
			if k+1 == len(hostNameList) {
				break CbLoop
			}
			k++
		}
	}

	return hostNameList
}

// manageReservations calls the appropriate reservation management function to operate on the given time parameter.
func manageReservations(ct *time.Time, m func(*time.Time) error) error {
	return m(ct)
}

// closeoutReservations will delete expired reservations that have ended up to the given time.
func closeoutReservations(checkTime *time.Time) error {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	timeParams := map[string]time.Time{"to-end": *checkTime}

	// get all reservations that expired on or before checkTime and delete them
	resList, err := dbReadReservationsTx(nil, timeParams)
	if err != nil {
		return err
	} else if len(resList) > 0 {

		logger.Info().Msgf("removing %d reservations: %v", len(resList), resNamesOfResList(resList))

		clusters, cErr := dbReadClustersTx(nil)
		if cErr != nil {
			logger.Error().Msgf("%v", cErr)
		}

		for _, r := range resList {

			logger.Debug().Msgf("begin removing reservation '%s'", r.Name)

			resClone := r.DeepCopy()

			// transaction to delete the reservation
			if err = performDbTx(func(tx *gorm.DB) error {
				// delete the reservation - this will uninstall from hosts, remove power perms,
				// set hosts back to available, and remove the res from the db
				_, err = doDeleteRes(&r, tx, true, &logger)
				return err
			}); err != nil {
				logger.Error().Msgf("failed to delete reservation '%s' - %v", r.Name, err)
				continue
			}

			if hErr := resClone.HistCallback(resClone, HrFinished); hErr != nil {
				logger.Error().Msgf("failed to record reservation '%s' finished to history", resClone.Name)
			}

			// notify user of expired reservation
			logger.Info().Msgf("reservation '%s' expired at %s -- deleting", resClone.Name, resClone.End.Format(common.DateTimeLongFormat))
			if expireEvent := makeResWarnNotifyEvent(EmailResExpire, 0, resClone, clusters[0].Name); expireEvent != nil {
				resNotifyChan <- *expireEvent
			}

			// uninstall reservation vlan and tftp
			if err = uninstallRes(resClone); err != nil {
				logger.Error().Msgf("%v", err)
			}

		}

	} else {
		logger.Debug().Msg("no reservations are expired")
	}

	return nil
}

// doMaintenance calls the appropriate maintenance management function to operate on the given time parameter.
func doMaintenance(ct *time.Time, m func(*time.Time) error) error {
	return m(ct)
}

// puts the host(s) of an ending reservation into a maintenance/reset period
// where the host(s) are made unavailable for the configured length of time.
// If a Distro is declared as a default, it will be installed to the
// reservation's hosts.
func startMaintenance(res *MaintenanceRes) error {
	// get the admin user
	admin, _, err := getIgorAdminTx()
	if err != nil {
		return fmt.Errorf("error retrieving igor-admin while starting maintenance - %v", err.Error())
	}
	now := time.Now()
	maintnanceEnd := res.MaintenanceEndTime
	maintenanceDuration := maintnanceEnd.Sub(now)
	logger.Debug().Msgf("reservation %v going into maintenenace mode from %v to %v (duration: %v).", res.ReservationName, now, maintnanceEnd, maintenanceDuration)

	// turn all hosts to the unavailable state
	logger.Debug().Msgf("changing state of nodes for reservation %v to blocked", res.ReservationName)
	changes := map[string]interface{}{"State": HostBlocked}
	err = performDbTx(func(tx *gorm.DB) error {
		err := dbEditHosts(res.Hosts, changes, tx)
		if err != nil {
			logger.Error().Msg(err.Error())
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("error in maintenance changing hosts to blocked state - %v", err.Error())
	}

	// check for a default distro image
	hasDefaultDistro := false
	currentDefaultDistros, err := dbReadDistrosTx(map[string]interface{}{"is_default": true})
	if err != nil {
		logger.Error().Msgf("unexpected error searching for default distro during maintenance period of reservation %s", res.ReservationName)
		return err
	}
	if len(currentDefaultDistros) > 0 {
		if len(currentDefaultDistros) > 1 {
			logger.Error().Msgf("is_default returned %v results", len(currentDefaultDistros))
		}
		hasDefaultDistro = true
	}
	if hasDefaultDistro {
		cdd := currentDefaultDistros[0]
		// create a temp profile from the default distro image
		profile := &Profile{
			Name:   res.ReservationName + "_maintenance",
			Distro: cdd,
		}
		// create a temp reservation using the temp profile
		tempRes := &Reservation{
			Name:    res.ReservationName + "_maintenance",
			Owner:   *admin,
			Hosts:   res.Hosts,
			Profile: *profile,
		}
		// install the default image to all res's hosts
		igor.IResInstaller.Install(tempRes)

		// power on the hosts
		logger.Debug().Msgf("power cycling hosts for reservation '%s'", tempRes.Name)
		if _, powerErr := doPowerHosts(PowerCycle, namesOfHosts(tempRes.Hosts), &logger); powerErr != nil {
			// don't return this error we still want to mark it installed
			logger.Error().Msgf("problem powering cycling hosts for reservation '%s': %v", tempRes.Name, powerErr)
		}
	}
	return nil
}

func finishMaintenance(now *time.Time) error {
	mReses, err := dbGetMaintenanceRes()
	if err != nil {
		logger.Error().Msgf("error getting maintenance Reservation list, aborting start process")
		return err
	}
	// get admin user
	admin, _, _ := getIgorAdminTx()
	for _, res := range mReses {
		if now.After(res.MaintenanceEndTime) {
			logger.Debug().Msgf("reservation %v going out of maintenenace mode.", res.ReservationName)
			hosts := res.Hosts
			// make sure no hosts are currently engaged in an active reservation
			// if they are, exclude them from the finish maintenance process
			activeHosts := hasActiveReservations(hosts)
			for _, host := range activeHosts {
				hosts = removeHost(hosts, &host)
			}

			// prepare a temp reservation
			// create a temp profile from the image
			profile := &Profile{
				Name: res.ReservationName + "_maintenance",
			}
			// create a new res based on default distro
			tempRes := &Reservation{
				Name:    res.ReservationName + "_maintenance",
				Owner:   *admin,
				Hosts:   hosts,
				Profile: *profile,
			}
			hasDefaultDistro := false
			// check for a default distro image
			currentDefaultDistros, err := dbReadDistrosTx(map[string]interface{}{"is_default": true})
			if err != nil {
				logger.Error().Msgf("unexpected error searching for default distro during maintenance period of reservation %s", res.ReservationName)
				return err
			}
			if len(currentDefaultDistros) > 0 {
				if len(currentDefaultDistros) > 1 {
					logger.Error().Msgf("is_default returned %v results", len(currentDefaultDistros))
				}
				hasDefaultDistro = true
			}
			if hasDefaultDistro {
				// power off the hosts
				logger.Debug().Msgf("powering off hosts for reservation '%s'", tempRes.Name)
				if _, powerErr := doPowerHosts(PowerOff, namesOfHosts(tempRes.Hosts), &logger); powerErr != nil {
					// don't return this error we still want to mark it installed
					logger.Error().Msgf("problem powering off hosts for reservation '%s': %v", tempRes.Name, powerErr)
				}

				// uninstall the default image from the res hosts
				igor.IResInstaller.Uninstall(tempRes)

			}

			// turn all hosts back to an available state
			logger.Debug().Msgf("changing state of nodes for reservation %v to available", tempRes.Name)
			changes := map[string]interface{}{"State": HostAvailable}
			_ = performDbTx(func(tx *gorm.DB) error {
				err := dbEditHosts(tempRes.Hosts, changes, tx)
				if err != nil {
					logger.Error().Msg(err.Error())
				}
				return err
			})
			// remove the res from db table
			if err := dbDeleteMaintenanceRes(&res); err != nil {
				logger.Error().Msgf("error deleting MaintenanceRes %v - %v", res.ReservationName, err.Error())
			}
		}
	}
	return nil
}

// installReservations will install any reservation up to the given time provided it hasn't already been installed.
func installReservations(checkTime *time.Time) error {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	// now look for any reservations that are starting around the check time
	timeParams := map[string]time.Time{"to-start": *checkTime}
	resList, err := dbReadReservationsTx(nil, timeParams)
	if err != nil {
		return err
	} else if len(resList) > 0 {
		for _, r := range resList {
			if !r.Installed {
				// sanity check that the hosts having their state updated should be HOST_AVAILABLE (0)
				for _, h := range r.Hosts {
					if h.State > HostAvailable {
						logger.Error().Msgf("host %s for reservation '%s' start in the state %v before being made available", h.Name, r.Name, h.State)
					}
				}

				if err = performDbTx(func(tx *gorm.DB) error {

					// change the reservation's hosts to 'reserved'
					logger.Debug().Msg("changing state of reservation hosts to reserved")
					changes := map[string]interface{}{"State": HostReserved}
					if ehErr := dbEditHosts(r.Hosts, changes, tx); ehErr != nil {
						return ehErr
					}

					// create the power permission for the reservation's hosts and add it to the permissions table
					logger.Debug().Msgf("activating power permissions for reservation %s", r.Name)
					powerPerm, permErr := NewPermission(makeNodePowerPerm(r.Hosts))
					if permErr != nil {
						return permErr
					}

					if apErr := dbAppendPermissions(&r.Group, []Permission{*powerPerm}, tx); apErr != nil {
						return apErr
					}

					// skip if not using vlan
					if igor.Vlan.Network != "" {
						// update network config
						if nsErr := networkSet(r.Hosts, r.Vlan); nsErr != nil {
							return fmt.Errorf("error setting network isolation: %v", nsErr)
						}
					}

					// install the reservation's profile to its hosts
					logger.Debug().Msgf("installing PXE files for reservation %s", r.Name)
					if irErr := igor.IResInstaller.Install(&r); irErr != nil {
						// update the reservation with the error message
						if irErr = dbEditReservation(&r, map[string]interface{}{"install_error": irErr.Error()}, tx); irErr != nil {
							return irErr
						}
						return irErr
					}

					if r.CycleOnStart {
						logger.Debug().Msgf("power cycling hosts for reservation '%s'", r.Name)
						if _, powerErr := doPowerHosts(PowerCycle, namesOfHosts(r.Hosts), &logger); powerErr != nil {
							// don't return this error we still want to mark it installed
							logger.Error().Msgf("problem powering cycling hosts for reservation '%s': %v", r.Name, powerErr)
						}
					} else {
						logger.Warn().Msgf("The reservation '%s' was not powered cycled at start", r.Name)
					}

					// update the reservation as installed
					return dbEditReservation(&r, map[string]interface{}{"installed": true}, tx)

				}); err != nil {
					logger.Error().Msgf("failed to install reservation '%s' - %v", r.Name, err)
					continue
				}

				if hErr := r.HistCallback(&r, HrInstalled); hErr != nil {
					logger.Error().Msgf("failed to record historical change to reservation '%s'", r.Name)
				}

				clusters, cErr := dbReadClustersTx(nil)
				if cErr != nil {
					return cErr
				}

				if startEvent := makeResWarnNotifyEvent(EmailResStart, 0, r.DeepCopy(), clusters[0].Name); startEvent != nil {
					resNotifyChan <- *startEvent
				}
			}
		}
	} else {
		logger.Debug().Msg("no reservations are starting")
	}

	return nil
}

// sendExpirationWarnings will check if any reservation at the given time is due to get a warning email and
// dispatch an event to the notification manager if true.
func sendExpirationWarnings(checkTime *time.Time) error {

	// For sending out reservation expiration warnings
	if *igor.Config.Email.ResNotifyOn {

		maxWindow := ResNotifyTimes[len(ResNotifyTimes)-1]
		timeParams := map[string]time.Time{"to-end": checkTime.Add(maxWindow)}
		resList, err := dbReadReservationsTx(nil, timeParams)
		if err != nil {
			return err
		}

		clusters, cErr := dbReadClustersTx(nil)
		if cErr != nil {
			return cErr
		}

		now := time.Now()
		for _, r := range resList {
			for i := 0; i < len(ResNotifyTimes); i++ {

				var resWarnEvent *ResNotifyEvent
				timeLeft := r.End.Sub(now) // amount of time left in res

				if i == 0 && timeLeft <= ResNotifyTimes[0] && r.NextNotify >= ResNotifyTimes[0] {
					resWarnEvent = makeResWarnNotifyEvent(EmailResFinalWarn, 0, r.DeepCopy(), clusters[0].Name)
				} else if i > 0 && ResNotifyTimes[i-1] < timeLeft && timeLeft <= ResNotifyTimes[i] && r.NextNotify >= ResNotifyTimes[i] {
					resWarnEvent = makeResWarnNotifyEvent(EmailResWarn, ResNotifyTimes[i-1], r.DeepCopy(), clusters[0].Name)
				}

				if resWarnEvent != nil {
					resNotifyChan <- *resWarnEvent
					logger.Debug().Msgf("reservation '%s' has pending expiration less than or equal to %s", r.Name, ResNotifyTimes[i].String())
					break
				}
			}
		}
	}

	return nil
}
