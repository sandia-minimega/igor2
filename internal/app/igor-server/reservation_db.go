// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"igor2/internal/pkg/common"
)

// dbCreateReservation puts a new reservation into the database.
func dbCreateReservation(res *Reservation, tx *gorm.DB) error {

	// if res has a new default profile, create first in db
	if res.Profile.IsDefault {
		if rErr := dbCreateProfile(&res.Profile, tx); rErr != nil {
			return rErr
		}
	}

	oPerms, err := createResOwnerPerms(res.Name)
	if err != nil {
		return err
	}
	pug, pugErr := res.Owner.getPug()
	if pugErr != nil {
		return pugErr
	}
	if err = dbAppendPermissions(pug, oPerms, tx); err != nil {
		return err
	}
	gPerms, gErr := createResGroupPerms(res)
	if gErr != nil {
		return gErr
	}
	if err = dbAppendPermissions(&res.Group, gPerms, tx); err != nil {
		return err
	}

	result := tx.Create(&res)
	return result.Error
}

// dbReadReservationsTx finds all reservations matching the query and time parameters passed to it with a new transaction.
func dbReadReservationsTx(queryParams map[string]interface{}, timeParams map[string]time.Time) (resList []Reservation, err error) {

	err = performDbTx(func(tx *gorm.DB) error {
		resList, err = dbReadReservations(queryParams, timeParams, tx)
		return err
	})

	return resList, err
}

// dbReadReservations finds all reservations matching the query and time parameters passed to it within an existing transaction.
func dbReadReservations(queryParams map[string]interface{}, timeParams map[string]time.Time, tx *gorm.DB) (resList []Reservation, err error) {

	// if no params given, return all reservations
	if len(queryParams) == 0 && len(timeParams) == 0 {
		result := tx.Joins("Owner").Joins("Group").Joins("Profile").
			Preload("Profile.Distro").Preload("Profile.Distro.DistroImage").Preload("Profile.Distro.Kickstart").Preload("Profile.Owner").Preload("Profile.Owner.Groups").
			Preload("Owner.Groups").Preload("Hosts").Find(&resList)
		return resList, result.Error
	}

	tx = tx.Preload("Owner").Preload("Group").Preload("Profile").
		Preload("Profile.Distro").Preload("Profile.Distro.DistroImage").Preload("Profile.Distro.Kickstart").Preload("Profile.Owner").Preload("Profile.Owner.Groups").
		Preload("Owner.Groups").Preload("Hosts")

	if len(timeParams) > 0 {
		resolveTimeWhereClauses(timeParams, tx)
	}

	for key, val := range queryParams {
		if strings.HasPrefix(key, "x-") { // skip comparison parameters
			continue
		} else {
			switch val.(type) {
			case string, bool, int:
				tx = tx.Where(key, val)
			case []int:
				if strings.ToLower(key) == "hosts" {
					tx = tx.Joins("JOIN reservations_hosts ON reservations_hosts.reservation_id = ID AND host_id IN ?", val)
				} else if strings.ToLower(key) == "distro_id" {
					tx = tx.Joins("JOIN profiles ON reservations.profile_id = profiles.id").Where("profiles.distro_id IN ?", val)
				} else {
					tx = tx.Where(key+" IN ?", val)
				}
			case []string:
				tx = tx.Where(key+" IN ?", val)
			default:
				logger.Error().Msgf("dbReadReservations: incorrect parameter type %T received for %s: %v", val, key, val)
			}
		}
	}

	result := tx.Find(&resList)
	return resList, result.Error
}

func dbEditReservation(res *Reservation, changes map[string]interface{}, tx *gorm.DB) error {

	// Change the name of the reservation
	if name, ok := changes["Name"].(string); ok {
		if perms, pResultErr := dbGetPermissionsByName(PermReservations, res.Name, tx); pResultErr != nil {
			return pResultErr
		} else {
			oldName := PermDividerToken + res.Name + PermDividerToken
			newName := PermDividerToken + name + PermDividerToken
			for _, p := range perms {
				newFact := strings.Replace(p.Fact, oldName, newName, 1)
				if result := tx.Model(&p).Update("Fact", newFact); result.Error != nil {
					return result.Error
				}
			}
			if result := tx.Model(&res).Update("Name", name); result.Error != nil {
				return result.Error
			}
			delete(changes, "Name")
		}
	}

	// change ownership of the reservation
	if _, ok := changes["OwnerID"]; ok {
		pList := changes["owner-perms"].([]Permission)
		if result := tx.Model(pList).Update("GroupID", changes["p-owner-gid"].(int)); result.Error != nil {
			return result.Error
		}
		delete(changes, "owner-perms")
		delete(changes, "p-owner-gid")
	}

	// change the group associated with the reservation
	if _, ok := changes["GroupID"]; ok {
		pList := changes["group-perms"].([]Permission)
		if result := tx.Model(pList).Update("GroupID", changes["p-gid"].(int)); result.Error != nil {
			return result.Error
		}
		delete(changes, "group-perms")
		delete(changes, "p-gid")
	}

	// change the reservation profile
	if profile, ok := changes["profile"].(*Profile); ok {
		if newProfile, ok := changes["create_new_profile"].(bool); ok && newProfile {
			if rErr := dbCreateProfile(profile, tx); rErr != nil {
				return rErr
			}
			delete(changes, "create_new_profile")
		}
		// if the old profile is a default, then destroy it
		if res.Profile.IsDefault {
			if rErr := dbDeleteProfile(&res.Profile, tx); rErr != nil {
				return rErr
			}
		}
		res.Profile = *profile
		tx.Save(&res)
		delete(changes, "profile")
	}

	// user wanted to modify the default profile's kernel
	if pk, ok := changes["profile_kernel"].(string); ok {
		if err := dbEditProfile(&res.Profile, map[string]interface{}{"kernel_args": pk}, tx); err != nil {
			return fmt.Errorf("unable to modify temp profile")
		}
		delete(changes, "profile_kernel")
	}

	// do drop only
	if dropHosts, ok := changes["dropHosts"].([]Host); ok {

		// if this reservation is current we need to update the power permissions and change the
		// dropped hosts' states to available
		if _, ok = changes["resIsNow"].(bool); ok {

			var result *gorm.DB

			for _, dropHost := range dropHosts {
				if dropHost.State != HostBlocked {
					result = tx.Model(dropHost).Update("State", HostAvailable)
					if result.Error != nil {
						return result.Error
					}
				}
			}

			p := changes["pUpdate"].(Permission)
			result = tx.Model(&Permission{}).Where("id = ?", p.ID).Update("Fact", p.Fact)
			if result.Error != nil {
				return result.Error
			}
		}

		// in any case, drop the appropriate entries from the reservations_hosts lookup table
		if clErr := tx.Model(&res).Association("Hosts").Delete(dropHosts); clErr != nil {
			return clErr
		}

		return nil
	}

	// do add only
	if addHosts, ok := changes["addHosts"].([]Host); ok {
		// add the appropriate entries from the reservations_hosts lookup table
		if clErr := tx.Model(&res).Association("Hosts").Append(addHosts); clErr != nil {
			return clErr
		}
		return nil
	}

	// change the rest of the fields, if any
	var fields []string
	for k := range changes {
		fields = append(fields, k)
	}
	if result := tx.Model(&res).Select(fields).Updates(changes); result.Error != nil {
		return result.Error
	}

	return nil
}

func dbDeleteReservation(res *Reservation, perms []Permission, isResNow bool, tx *gorm.DB) error {

	// if this reservation is currently running or already finished (we are cleaning up after a prolonged shutdown),
	// change state of the reservation hosts back to 'available'
	if isResNow {

		for _, host := range res.Hosts {
			if host.State != HostBlocked {
				result := tx.Model(&host).Omit("access_group_id").Update("State", HostAvailable)
				if result.Error != nil {
					return result.Error
				}
			}
		}
	}

	// delete the associations with the hosts table
	if clErr := tx.Model(&res).Association("Hosts").Clear(); clErr != nil {
		return clErr
	}

	// delete the permissions for this reservation
	result := tx.Delete(perms)
	if result.Error != nil {
		return result.Error
	}

	// if the profile attached to this res is a default, destroy it
	if res.Profile.IsDefault {
		if err := dbDeleteProfile(&res.Profile, tx); err != nil {
			return err
		}
	}

	// delete the reservation
	result = tx.Delete(&res)
	return result.Error
}

// Generates additional clauses to the reservation search params that narrow results
// by time instances and ranges.
func resolveTimeWhereClauses(timeParams map[string]time.Time, tx *gorm.DB) {

	var fromStart *time.Time
	var toEnd *time.Time
	var fromEnd *time.Time
	var toStart *time.Time

	for key, val := range timeParams {
		switch key {
		case "from-start":
			t := val
			fromStart = &t
		case "to-end":
			t := val
			toEnd = &t
		case "from-end":
			t := val
			fromEnd = &t
		case "to-start":
			t := val
			toStart = &t
		default:
		}
	}

	// reservations that started on-or-after t1 and ended on-or-before t2
	if fromStart != nil && toEnd != nil {
		tx = tx.Where("start >= ? AND end <= ?", fromStart, toEnd)
	} else if fromStart != nil && toStart != nil {
		// reservations that started between t1 and t2 (inclusive)
		tx = tx.Where("start BETWEEN ? AND ?", fromStart, toStart)
	} else if fromEnd != nil && toEnd != nil {
		// reservations that ended between t1 and t2 (inclusive)
		tx = tx.Where("end BETWEEN ? AND ?", fromEnd, toEnd)
	} else if toStart != nil && fromEnd != nil {
		// reservations that start on-or-before t1 and end on-or-after t2 (inclusive)
		tx = tx.Where("start <= ? AND end >= ?", toStart, fromEnd)
	} else if fromStart != nil {
		// reservations that started on-or-after t
		tx = tx.Where("start >= ?", fromStart)
	} else if toStart != nil {
		// reservations that started on-or-before t
		tx = tx.Where("start <= ?", toStart)
	} else if toEnd != nil {
		// reservations that ended on-or-before t
		tx = tx.Where("end <= ?", toEnd)
	} else if fromEnd != nil {
		// reservations that ended on-or-after t
		tx = tx.Where("end >= ?", fromEnd)
	}

}

func createResGroupPerms(res *Reservation) ([]Permission, error) {
	psList := makeResGroupPermStrings(res)
	var groupPerms []Permission
	for _, ps := range psList {
		p, err := NewPermission(ps)
		if err != nil {
			return groupPerms, err
		}
		groupPerms = append(groupPerms, *p)
	}
	return groupPerms, nil
}

func createResOwnerPerms(resvName string) ([]Permission, error) {
	// the owner only needs 'edit:*' permissions. Delete his covered by the group.
	pstr := NewPermissionString(PermReservations, resvName, PermEditAction, PermWildcardToken)
	ownerResvEdit, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	return []Permission{*ownerResvEdit}, nil
}

// dbCheckResvConflicts scans the database for reservations that conflict with the given slice of host names in the interval
// specified by the starTime and endTime. A reservation should be good to schedule if the status response is 200/OK. Returns:
//
//	nil,200,nil if no conflicts were found.
//	list,409,err if one or more reservations were found that overlap the specified input.
//	nil,500,err if there was an internal problem.
func dbCheckResvConflicts(hosts []string, startTime, endTime time.Time, tx *gorm.DB) ([]Reservation, int, error) {

	var result *gorm.DB
	var resList []Reservation
	resetEndTime := determineNodeResetTime(endTime)
	// Find reservations on each declared node that overlap with the proposed time slot
	// Reject if an existing reservation is found where:
	//  - the proposed start time overlaps (the reservation is already running on the node when the new res would start)
	//  - the proposed end time overlaps (the reservation is scheduled to start on the node before the new reservation would end)
	//  - a reservation starts and ends inside the time interval of the proposed reservation
	result = tx.Table("reservations r, hosts h").
		Select("r.*").
		Joins("INNER JOIN reservations_hosts rh ON r.id = rh.reservation_id AND h.id = rh.host_id").
		Where("h.name IN ? AND ((r.start <= ? AND ? < r.reset_end) OR (r.start < ? AND ? <= r.reset_end) OR (? <= r.start AND r.reset_end <= ?))",
			hosts, startTime, startTime, resetEndTime, resetEndTime, startTime, resetEndTime).Scan(&resList)

	if result.Error != nil {
		return nil, http.StatusInternalServerError, result.Error
	} else if result.RowsAffected > 0 {
		return resList, http.StatusConflict, fmt.Errorf("found existing reservation(s) on node(s) conflicting with time interval [%v, %v]",
			startTime.Format(common.DateTimeLongFormat), endTime.Format(common.DateTimeLongFormat))
	}
	return nil, http.StatusOK, nil
}

// ReservationSlot matches the data types that get pulled back from dbFindOpenSlots query
// This may seem a bit hacky since we ultimately want ReservationTimeSlot, but I can't figure
// out how to pull the query results back from SQLite/GORM without the time fields being text.
type ReservationSlot struct {
	Hostname       string
	Hostnum        int
	ResName        string
	ResStart       string
	AvailSlotBegin string
	NextResName    string
	AvailSlotEnd   string
}

// ReservationTimeSlot is the target for ReservationSlot objects that convert string time fields to time.Time fields.
type ReservationTimeSlot struct {
	Hostname       string
	Hostnum        int
	ResName        string
	ResStart       time.Time
	AvailSlotBegin time.Time
	NextResName    string
	AvailSlotEnd   time.Time
}

// dbFindOpenSlots queries the database for time periods from now through the max allowable ending datetime that are
// at least of length durNeeded for all hosts named in hostNameList.
//
// The method will favor unused nodes to satisfy the reservation request. This helps
// to spread out reservations in a way that most nodes will see usage at some point at the expense of contiguous blocks
// being allocated. It will also mean fewer instances of users being unable to extend reservations when the cluster
// has sparse number of future reservations.
//
// This is purely finding all time windows that meet the size requirement. Results need to be filtered.
func dbFindOpenSlots(hostNameList []string, startTime time.Time, durNeeded time.Duration, maxEnd time.Time, numHostsReq int, tx *gorm.DB) ([]ReservationTimeSlot, int, error) {

	// use max end time of last minute of the year that is 25 years from now
	resDurMinutes := strconv.Itoa(int(durNeeded.Minutes()))

	const openSlotsSQL = `
WITH
    -- slots on nodes with no reservations
    free_slots AS (
        SELECT
            h.name           AS hostname,
            h.sequence_id    AS hostnum,
            NULL             AS res_name,
            NULL             AS res_start,
            ?                AS avail_slot_begin,
            NULL             AS next_res_name,
            ?                AS avail_slot_end
        FROM hosts h
             LEFT JOIN reservations_hosts rh
                  ON rh.host_id = h.id
        WHERE
            rh.host_id IS NULL
          AND h.state   < ?
          AND h.name IN (?)
    ),

    -- slots on nodes after last reservation
    last_res_slots AS (
        SELECT
            h.name           AS hostname,
            h.sequence_id    AS hostnum,
            r.name           AS res_name,
            r.start          AS res_start,
            r.reset_end      AS avail_slot_begin,
            NULL             AS next_res_name,
            ?                AS avail_slot_end
        FROM hosts h
            JOIN (
                SELECT rh.host_id, MAX(r.start) AS max_start
                FROM reservations r
                    JOIN reservations_hosts rh
                        ON r.id = rh.reservation_id
                GROUP BY rh.host_id
            ) lr
                ON lr.host_id = h.id
              JOIN reservations r
                ON r.start = lr.max_start
        WHERE
            h.state < ?
          AND h.name IN (?)
    ),

    -- slots on nodes with a large enough gap between reservations
    gap_slots AS (
        SELECT
            h.name           AS hostname,
            h.sequence_id    AS hostnum,
            l.name           AS res_name,
            l.start          AS res_start,
            l.reset_end      AS avail_slot_begin,
            r.name           AS next_res_name,
            r.start          AS avail_slot_end
        FROM reservations l
            JOIN reservations_hosts rhl
                 ON l.id = rhl.reservation_id
            JOIN hosts h
                 ON h.id = rhl.host_id
            JOIN reservations r
                 ON r.id = (
                     SELECT r2.id
                     FROM reservations r2
                         JOIN reservations_hosts rh2
                              ON r2.id = rh2.reservation_id 
                                 AND rh2.host_id = h.id
                     WHERE DATETIME(l.reset_end, '+'||?||' minutes') < DATETIME(r2.start)
                     ORDER BY r2.start
                     LIMIT 1
                 )
        WHERE
            h.state < ?
          AND h.name IN (?)
          AND NOT EXISTS (
            SELECT 1
            FROM reservations x
                JOIN reservations_hosts rxi
                     ON x.id = rxi.reservation_id
                         AND rxi.host_id = h.id
            WHERE l.reset_end < x.start
              AND x.start   < r.start
        )
    ),

    free_count AS (
        SELECT COUNT(*) AS cnt
        FROM free_slots
    )

-- final query prioritizes using unused nodes
-- unless there are not enough available

SELECT * FROM free_slots
WHERE (SELECT cnt FROM free_count) >= ?

UNION ALL

SELECT * FROM
     (
        SELECT * FROM free_slots
        UNION ALL
        SELECT * FROM last_res_slots
        UNION ALL
        SELECT * FROM gap_slots
     ) all_slots
WHERE (SELECT cnt FROM free_count) < ?

ORDER BY hostnum, avail_slot_begin;
`
	var slots []ReservationSlot
	var timeSlotListAll []ReservationTimeSlot

	err := tx.Raw(
		openSlotsSQL,
		// free_slots placeholders:
		startTime, maxEnd, HostBlocked, hostNameList,
		// last_res_slots placeholders:
		maxEnd, HostBlocked, hostNameList,
		// gap_slots placeholders:
		resDurMinutes, HostBlocked, hostNameList,
		// gating placeholders (numHostsReq twice):
		numHostsReq, numHostsReq,
	).Scan(&slots).Error
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	timeSlotListAll = convertToTimeSlotSlice(slots)
	sortTimeSlots(timeSlotListAll)
	return timeSlotListAll, http.StatusOK, nil
}

// sorts time slot values by earliest available begin time and then, if that field is
// equal, by earliest available end time, and if those fields are equal to then sort in order
// by node sequence number.
func sortTimeSlots(timeSlots []ReservationTimeSlot) {
	sort.Slice(timeSlots, func(i, j int) bool {
		if timeSlots[i].AvailSlotBegin.Before(timeSlots[j].AvailSlotBegin) {
			return true
		}
		if timeSlots[i].AvailSlotBegin.After(timeSlots[j].AvailSlotBegin) {
			return false
		}
		if timeSlots[i].AvailSlotEnd.Before(timeSlots[j].AvailSlotEnd) {
			return true
		}
		if timeSlots[i].AvailSlotEnd.After(timeSlots[j].AvailSlotEnd) {
			return false
		}
		return timeSlots[i].Hostnum < timeSlots[j].Hostnum
	})
}

func convertToTimeSlotSlice(slots []ReservationSlot) []ReservationTimeSlot {

	var timeSlots []ReservationTimeSlot

	for _, s := range slots {
		timeSlot := copySlotToTimeSlot(s)
		timeSlots = append(timeSlots, *timeSlot)
	}

	return timeSlots
}

func copySlotToTimeSlot(slot ReservationSlot) *ReservationTimeSlot {

	var TimeFormatDb = "2006-01-02 15:04:05Z07:00"
	var TimeFormatZoneDb = "2006-01-02 15:04:05.999999999Z07:00"

	var err error
	var tasb, tase, resStart time.Time

	if tasb, err = time.Parse(TimeFormatDb, slot.AvailSlotBegin); err != nil {
		if tasb, err = time.Parse(TimeFormatZoneDb, slot.AvailSlotBegin); err != nil {
			if tasb, err = time.Parse(time.RFC3339Nano, slot.AvailSlotBegin); err != nil {
				if tasb, err = time.Parse(time.RFC3339, slot.AvailSlotBegin); err != nil {

				}
			}
		}
	}

	if tase, err = time.Parse(TimeFormatDb, slot.AvailSlotEnd); err != nil {
		if tase, err = time.Parse(TimeFormatZoneDb, slot.AvailSlotEnd); err != nil {
			if tase, err = time.Parse(time.RFC3339Nano, slot.AvailSlotEnd); err != nil {
				if tase, err = time.Parse(time.RFC3339, slot.AvailSlotEnd); err != nil {

				}
			}
		}
	}

	if resStart, err = time.Parse(TimeFormatDb, slot.ResStart); err != nil {
		if resStart, err = time.Parse(TimeFormatZoneDb, slot.ResStart); err != nil {
			if resStart, err = time.Parse(time.RFC3339Nano, slot.ResStart); err != nil {
				if resStart, err = time.Parse(time.RFC3339, slot.ResStart); err != nil {

				}
			}
		}
	}

	timeSlot := &ReservationTimeSlot{
		Hostname:       slot.Hostname,
		Hostnum:        slot.Hostnum,
		ResName:        slot.ResName,
		NextResName:    slot.NextResName,
		AvailSlotBegin: tasb,
		AvailSlotEnd:   tase,
		ResStart:       resStart,
	}

	return timeSlot
}
