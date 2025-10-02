// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
)

func doCreateReservation(resParams map[string]interface{}, r *http.Request) (res *Reservation, resIsNow bool, status int, err error) {

	clog := hlog.FromRequest(r)
	user := getUserFromContext(r)
	clog.Debug().Msgf("create reservation: by user %s, called with params %+v", user.Name, resParams)

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		resName := resParams["name"].(string)

		// If the reservation already exists, abort!
		if found, findErr := resvExists(resName, tx); findErr != nil {
			return findErr
		} else if found {
			status = http.StatusConflict
			return fmt.Errorf("reservation '%s' already exists", resName)
		}

		// assume the requesting user will be the reservation owner
		resOwner := getUserFromContext(r)

		// check if user is requesting as an admin
		isElevated := userElevated(resOwner.Name)

		// check if the owner param is set and permitted if different from the requesting user
		if ownerParam, ok := resParams["owner"].(string); ok {
			if ownerParam != "" && ownerParam != resOwner.Name {
				if isElevated {
					if users, guStatus, guErr := getUsers([]string{ownerParam}, true, tx); guErr != nil {
						status = guStatus
						return guErr
					} else {
						resOwner = &users[0]
					}
				} else {
					status = http.StatusBadRequest
					return fmt.Errorf("non-elevated users cannot specify a different owner for a reservation")
				}
			}
		}

		// does user want to add kernel args to the temp profile?
		kernelArgs, kOk := resParams["kernelArgs"].(string)

		// create the profile from either the given distro or profile name
		var profile *Profile
		if distroName, dOk := resParams["distro"].(string); dOk {
			distroList, distroStatus, distroErr := getDistros([]string{distroName}, tx)
			if distroErr != nil {
				status = distroStatus
				return distroErr
			}
			distro := &distroList[0]

			if !resOwner.isMemberOfAnyGroup(distro.Groups) {
				status = http.StatusForbidden
				return fmt.Errorf("%s does not have access to distro '%s'", resOwner.Name, distro.Name)
			}
			newProfileName := generateDefaultProfileName(resOwner)
			profile = &Profile{
				Name:        newProfileName,
				Owner:       *resOwner,
				Distro:      *distro,
				IsDefault:   true,
				Description: "Default profile for distro " + distro.Name + " for reservation " + resName,
			}

			if kOk {
				profile.KernelArgs = kernelArgs
			}

		} else if profileName, pOk := resParams["profile"].(string); pOk {
			profileList, profileErr := dbReadProfiles(map[string]interface{}{"name": profileName, "owner_id": resOwner.ID}, tx)
			if profileErr != nil {
				return profileErr // uses default err status
			} else if len(profileList) == 0 {
				status = http.StatusConflict
				return fmt.Errorf("no profiles for user %v match name %v", profileName, resOwner.Name)
			}
			profile = &profileList[0]
			if profile.IsDefault {
				return fmt.Errorf("cannot use a temp profile in more than 1 reservation. Make the profile permanent first by editing its name, then try again")
			}
			// make sure the distro of this profile is still accessible to the user
			if dList, _, err := getDistros([]string{profile.Distro.Name}, tx); err != nil {
				return err
			} else if len(dList) == 0 {
				return fmt.Errorf("no distro returned with name %v from specified profile %v", profile.Distro.Name, profileName)
			} else {
				profDistro := &dList[0]
				if !resOwner.isMemberOfAnyGroup(profDistro.Groups) {
					return fmt.Errorf("%s does not currently have access to distro '%s' in profile '%s'", res.Owner.Name, profDistro.Name, profileName)
				}
			}

			if kOk {
				return fmt.Errorf("kernel args cannot be added to an existing profile when creating a new reservation -- edit the profile first")
			}
		} else {
			// we got neither a profile nor a distro?
			status = http.StatusNotFound
			return fmt.Errorf("must have either a distro or profile to create a reservation")
		}

		// The default reservation group is the owner's private group
		group, pugErr := resOwner.getPug()
		if pugErr != nil {
			return pugErr
		}

		// Check if the user specified a group.
		if groupName, ok := resParams["group"].(string); ok {
			if groupName == GroupNoneAlias {
				// user explicitly wants no res group. should be pug by default,
				// group already set to the user's pug directly above.
			} else if groupName == GroupAll {
				status = http.StatusBadRequest
				return fmt.Errorf("reservations cannot be assigned to the '%s' group", GroupAll)
			} else {
				groups, ggStatus, ggErr := getGroups([]string{groupName}, true, tx)
				if ggErr != nil {
					status = ggStatus
					return ggErr
				}
				group = &groups[0]
				// make sure the owner is also a member of the group specified
				if !resOwner.isMemberOfGroup(group) {
					return fmt.Errorf("user is not a member of group '%s'", groupName)
				}
			}
		}

		// Set the hosts - these are just place-holder or shell hosts for now
		// proper host scheduling is done below
		var hostNames []string
		var hosts []Host
		thisNodeList, nlOk := resParams["nodeList"].(string)
		if nlOk {
			if thisNodeList != "" {
				hostNames = igor.splitRange(thisNodeList)
				if hList, ghStatus, ghErr := getHosts(hostNames, true, tx); ghErr != nil {
					status = ghStatus
					return ghErr
				} else {
					hosts = hList
				}
			}
		}

		// validation should enforce that nodeList OR nodeCount is present, not both
		thisNodeCount, ncOk := resParams["nodeCount"].(float64)
		if ncOk {
			if thisNodeCount < 1 {
				status = http.StatusBadRequest
				return fmt.Errorf("reservation must include at least one host")
			}
			hosts = make([]Host, int(thisNodeCount))
		}

		// Check against allowed host max limit when not an elevated admin
		if !isElevated && igor.Scheduler.NodeReserveLimit > 0 && len(hosts) > igor.Scheduler.NodeReserveLimit {
			err = fmt.Errorf("only admins can make a reservation of more than %v nodes", igor.Scheduler.NodeReserveLimit)
			clog.Warn().Msgf("%v", err)
			status = http.StatusForbidden
			return err
		}

		// determine start and end times, and whether reservation starts immediately
		var resStart time.Time
		var resEnd time.Time

		if startTs, stOK := resParams["start"].(float64); stOK {
			start := time.Unix(int64(startTs), 0)
			resStart, resIsNow, err = evaluateResStartTime(start)
		} else {
			resStart, resIsNow, err = evaluateResStartTime(time.Time{})
		}
		if err != nil {
			status = http.StatusBadRequest
			return err
		}

		fDur, fOk := resParams["duration"].(float64)
		sDur, sOk := resParams["duration"].(string)

		if !fOk && !sOk {
			sDur = strconv.FormatInt(igor.Scheduler.DefaultReserveTime, 10) + "m"
			sOk = true
		}

		if fOk {
			resEnd = time.Unix(int64(fDur), 0)
			if !meetsMinResDuration(resEnd.Sub(resStart)) {
				status = http.StatusBadRequest
				err = fmt.Errorf("reservation duration must be larger than minimum value %v minutes", igor.Scheduler.MinReserveTime)
				return err
			}
		} else if sOk {
			dur, _ := common.ParseDuration(sDur)
			if !meetsMinResDuration(dur) {
				status = http.StatusBadRequest
				err = fmt.Errorf("reservation duration must be larger than minimum value %v minutes", igor.Scheduler.MinReserveTime)
				return err
			}
			resEnd = resStart.Add(dur).Truncate(time.Minute) // drop any seconds in the value
		}

		if err = checkScheduleLimit(resEnd, isElevated); err != nil {
			status = http.StatusBadRequest
			return err
		}

		// determine reset/maintenance end time
		resetEnd := determineNodeResetTime(resEnd)

		// set the VLAN
		vlan := 0
		// skip if not using vlan
		if igor.Vlan.Network != "" {
			if thisVlan, ok := resParams["vlan"].(string); ok {
				// user wants a specific vlan
				if thisVlan != "" {
					vlanInt, pvStatus, pvErr := parseVLAN(thisVlan, *resOwner, tx)
					if pvErr != nil {
						status = pvStatus
						return pvErr
					}
					vlan = vlanInt

				} else {
					status = http.StatusBadRequest
					return fmt.Errorf("vlan specified in reservation parameters, but no value included")
				}
			} else {
				// pick next available
				if vlan, err = nextVLAN(); err != nil {
					clog.Error().Msgf("error - %v", err.Error())
				}
			}
		}

		var cycleOnStart = true
		if noCycle, cOk := resParams["noCycle"].(bool); cOk && noCycle {
			cycleOnStart = false
			logger.Warn().Msgf(
				"the reservation '%s' was configured to not power cycle on start up by %s", resName, resOwner.Name)
		}

		// set next notification
		nextNotify := time.Duration(0)
		if *igor.Email.ResNotifyOn {
			now := time.Now()
			if resEnd.Sub(now) < ResNotifyTimes[0] {
				nextNotify = ResNotifyTimes[0]
			} else {
				for i := len(ResNotifyTimes) - 1; i >= 0; i-- {
					if resEnd.Sub(now) >= ResNotifyTimes[i] {
						nextNotify = ResNotifyTimes[i]
						break
					}
				}
			}
		} else {
			// set large in case notifications are turned on in future
			nextNotify = time.Hour * 24 * 365 * 5
		}

		// make hash identifier for change history
		var hashBytes []byte
		hashBytes = append(hashBytes, resName...)
		hashBytes = append(hashBytes, resOwner.Name...)
		hashBytes = append(hashBytes, group.Name...)
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(resStart.Unix()))
		hashBytes = append(hashBytes, b...)
		b = make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(resEnd.Unix()))
		hashBytes = append(hashBytes, b...)
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(vlan))
		hashBytes = append(hashBytes, b...)
		hash := sha1.New()
		hash.Write(hashBytes)

		// build reservation object
		res = &Reservation{
			Name:         resName,
			Owner:        *resOwner,
			Group:        *group,
			Start:        resStart,
			End:          resEnd,
			OrigEnd:      resEnd,
			ResetEnd:     resetEnd,
			Hosts:        hosts,
			Profile:      *profile,
			Vlan:         vlan,
			CycleOnStart: cycleOnStart,
			NextNotify:   nextNotify,
			Hash:         hex.EncodeToString(hash.Sum(nil)),
			HistCallback: doHistoryRecord,
		}

		// determine hosts to assign to reservation based on given host names or count requested
		if nlOk {
			if sbnStatus, sbnErr := scheduleHostsByName(res, tx, clog); sbnErr != nil {
				status = sbnStatus
				return sbnErr
			}
		} else {
			if hostList, sbaStatus, sbaErr := scheduleHostsByAvailability(res, tx, clog); sbaErr != nil {
				status = sbaStatus
				return sbaErr
			} else {
				res.Hosts = hostList
			}
		}
		// insert new reservation to the db
		return dbCreateReservation(res, tx)

	}); err != nil {
		return
	}

	if hErr := res.HistCallback(res, HrCreated); hErr != nil {
		clog.Error().Msgf("failed to record reservation '%s' create to history", res.Name)
	}

	return res, resIsNow, http.StatusCreated, nil
}

func parseVLAN(vlan string, user User, tx *gorm.DB) (int, int, error) {
	// First check to see if we've been handed a reservation name
	resList, err := dbReadReservations(map[string]interface{}{"name": vlan}, nil, tx)
	if err != nil {
		return -1, http.StatusInternalServerError, err
	}
	if len(resList) > 0 {
		// Check to see if resTarget owner is the same as user
		resTarget := resList[0]
		if resTarget.Owner.Name == user.Name {
			return resTarget.Vlan, http.StatusOK, nil
		}
		return -1, http.StatusForbidden, fmt.Errorf("owner of reservation specified for VLAN does not match user")
	}

	// See if it's a VLAN ID
	vlanID64, pErr := strconv.ParseInt(vlan, 10, 64)
	if pErr != nil {
		// It wasn't an int, either.
		return -1, http.StatusBadRequest, fmt.Errorf("expected VLAN to be reservation name or VLAN ID: %s", vlan)
	}
	vlanID := int(vlanID64)

	// Yep, it's an int
	if vlanID < igor.Vlan.RangeMin || vlanID > igor.Vlan.RangeMax {
		// VLAN number isn't in the permitted range
		return -1, http.StatusBadRequest, fmt.Errorf("VLAN number outside permitted range: %s", vlan)
	}

	// See who's already using that VLAN ID
	resList, err = dbReadReservations(map[string]interface{}{"vlan": vlan}, nil, tx)
	if err != nil {
		return -1, http.StatusInternalServerError, err
	} else if len(resList) > 0 {
		ownsOne := false
		for _, r := range resList {
			if r.Owner.Name == user.Name {
				ownsOne = true
			}
		}
		if !ownsOne {
			return -1, http.StatusForbidden, fmt.Errorf("cannot set VLAN -- must have ownership of at least one reservation using it: %s", vlan)
		}
	}
	return vlanID, http.StatusOK, nil
}

// Determines if reservation starts now or in the future and returns proper time values. If the future start date
// is less than 1 minute from the current local time, the reservation time is adjusted to start now.
func evaluateResStartTime(start time.Time) (resStart time.Time, resIsNow bool, err error) {

	var s time.Time

	now := time.Now()
	if start.IsZero() {
		start = now
	}
	interval := start.Sub(now)
	if interval < time.Minute {
		if interval < 0 {
			return resStart, false, fmt.Errorf("start time is in the past %s", start.Format(common.DateTimeCompactFormat))
		}
		s = now
		resIsNow = true
		resStart = time.Date(s.Year(), s.Month(), s.Day(), s.Hour(), s.Minute(), s.Second(), 0, time.Local)
	} else {
		s = start
		resIsNow = false
		resStart = time.Date(s.Year(), s.Month(), s.Day(), s.Hour(), s.Minute(), 0, 0, time.Local)
	}

	return
}
