// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

func showHandler(w http.ResponseWriter, r *http.Request) {
	// this is essentially the "landing" page for igor
	// generate the following data and return to the user:
	// - all active (current and future) reservations
	// - all hosts (including powered/unpowered)
	// - user's available profiles
	// - user's available distros
	clog := hlog.FromRequest(r)
	actionPrefix := "show"
	rb := common.NewResponseBodyShow()

	user := getUserFromContext(r)
	result, status, err := getShowData(user)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Debug().Msgf("%s success", actionPrefix)
	}
	rb.Data["show"] = result

	makeJsonResponse(w, status, rb)
}

func getShowData(user *User) (showData common.ShowData, code int, err error) {

	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		refreshPowerChan <- struct{}{}

		showData = common.ShowData{}

		reservations, rErr := dbReadReservations(nil, nil, tx)
		if rErr != nil {
			return rErr
		} else {
			showData.Reservations = filterReservationList(reservations, user)
		}
		hosts, hErr := dbReadHosts(nil, tx)
		if hErr != nil {
			return hErr
		} else if len(hosts) > 0 {
			showData.Hosts = filterHostList(hosts, nil, user)
			if clusters, cErr := dbReadClusters(map[string]interface{}{"id": hosts[0].ClusterID}, tx); cErr != nil {
				return cErr
			} else {
				showData.Cluster = clusters[0].getClusterData()
			}
		}

		profileParams := map[string]interface{}{}
		if !userElevated(user.Name) {
			// scope profiles to user-owned only if not elevated/admin
			profileParams = map[string]interface{}{"owner_id": user.ID}
		}
		profiles, pErr := dbReadProfiles(profileParams, tx)
		if pErr != nil {
			return pErr
		} else {
			showData.Profiles = filterProfileList(profiles)
		}
		distroParams := map[string]interface{}{}
		if !userElevated(user.Name) {
			// scope distros to user's allowed groups if not elevated/admin
			distroParams = map[string]interface{}{"groups": groupIDsOfGroups(user.Groups)}
		}
		distros, dErr := dbReadDistros(distroParams, tx)
		if dErr != nil {
			return dErr
		} else {
			showData.Distros = filterDistroList(distros)
		}

		groupNames := groupNamesOfGroups(user.Groups)
		for _, gn := range groupNames {
			if !(strings.HasPrefix(gn, GroupUserPrefix)) {
				showData.UserGroups = append(showData.UserGroups, gn)
			}
		}

		return nil
	}); err == nil {
		code = http.StatusOK
	}

	return
}

func publicShowHandler(w http.ResponseWriter, r *http.Request) {

	clog := hlog.FromRequest(r)
	actionPrefix := "public res data"
	status := http.StatusOK
	var err error
	var publicData string

	if igor.Server.AllowPublicShow {
		publicData, status, err = getPublicShowData()
	} else {
		status = http.StatusForbidden
		err = fmt.Errorf("%s has restricted igor reservation data from public view", igor.InstanceName)
	}

	if err != nil {
		if status >= http.StatusInternalServerError {
			clog.Error().Msgf("%s error - %v", actionPrefix, err)
		} else {
			clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
		}
		publicData = fmt.Sprintf("Status: %d\n%v\n", status, err)
	} else {
		clog.Debug().Msgf("%s success", actionPrefix)
	}

	w.Header().Set(common.ContentType, common.MTextPlain)
	w.WriteHeader(status)
	if _, err = w.Write([]byte(publicData)); err != nil {
		panic(err)
	}
}

func getPublicShowData() (publicData string, code int, err error) {

	code = http.StatusOK // default status, overridden at end if no errors

	resList, rErr := dbReadReservationsTx(nil, nil)
	if rErr != nil {
		err = rErr
		code = http.StatusInternalServerError
		return
	} else {

		publicData = "resName,owner,group,nodeCount,nodes,startTime,endTime\n"

		for _, r := range resList {

			sort.Slice(r.Hosts, func(i, j int) bool {
				return r.Hosts[i].SequenceID < r.Hosts[j].SequenceID
			})

			hostNameList := namesOfHosts(r.Hosts)

			// to assist with parsing comma-delimited fields
			hostRange := strings.Join(hostNameList, " ")

			var groupName string
			if !strings.HasPrefix(r.Group.Name, GroupUserPrefix) {
				groupName = r.Group.Name
			}

			resLine := make([]string, 7)
			resLine[0] = r.Name
			resLine[1] = r.Owner.Name
			resLine[2] = groupName
			resLine[3] = strconv.Itoa(len(r.Hosts))
			resLine[4] = hostRange
			resLine[5] = r.Start.Format(common.DateTimePublicFormat)
			resLine[6] = r.End.Format(common.DateTimePublicFormat)
			publicData += strings.Join(resLine, ",") + "\n"
		}
	}

	return
}
