// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// This section is for generating reports about Igor usage
func statsHandler(w http.ResponseWriter, r *http.Request) {
	// runs a stats command
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "stats"
	rb := common.NewResponseBody()

	result, status, err := runStats(queryMap)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success", actionPrefix)
	}
	rb.Data["stats"] = result

	makeJsonResponse(w, status, rb)
}

func runStats(optionParams map[string][]string) (stats common.StatsData, status int, err error) {
	option := "default"
	verbose := false
	// default stats window is 7 days from current time
	end := time.Now()
	start := end.Add(-(time.Hour * (24 * 7)))
	status = http.StatusInternalServerError
	err = nil

	// determine what type of stats to run and the time window
	if len(optionParams) > 0 {
		for k, v := range optionParams {
			switch k {
			case "option":
				// at the moment, we don't have any option besides default, so ignore this
			case "start":
				// try to parse time input to modify end variable
				if t, err := time.Parse("2006-Jan-02", v[0]); err == nil {
					end = t
				} else {
					logger.Debug().Msgf("error converting string %v to time.Time", v[0])
				}
			case "duration":
				// try to parse duration input
				logger.Debug().Msgf("changing stats duration from start %v", start)
				if d, err := strconv.Atoi(v[0]); err == nil {
					logger.Debug().Msgf("Atoi converted %s to %v", v[0], d)
					if d < 0 {
						msg := fmt.Sprintf("invalid value received for stats duration: %v", d)
						logger.Error().Msgf(msg)
						status = http.StatusBadRequest
						err = fmt.Errorf(msg)
						return stats, status, err
					} else if d == 0 {
						start = time.Time{}
					} else {
						start = end.AddDate(0, 0, -d)
					}

				} else {
					msg := fmt.Sprintf("error converting string %v to int", v[0])
					logger.Debug().Msgf(msg)
					status = http.StatusBadRequest
					return stats, status, fmt.Errorf(msg)
				}
			case "verbose":
				verbose = strings.ToLower(v[0]) == "true"
			}
		}
	}
	stats.Option = option
	stats.Start = start
	stats.End = end
	stats.Verbose = verbose

	var data []common.ResHistory
	// query test
	if err = performDbTx(func(tx *gorm.DB) error {
		result := tx.Table("history_records h").
			Select("h.hash AS hash, h.status AS status, h.name AS name, h.owner AS owner, h.profile AS profile, h.distro AS distro, h.vlan AS vlan, h.start AS start, h.end AS end, h.orig_end AS orig_end, h.extend_count AS extend_count, h.hosts AS hosts, h.created_at AS created_at").
			Order("h.created_at").
			Where("h.created_at >= ? AND h.created_at <= ?", start, end).
			Scan(&data)
		if result.Error != nil {
			return result.Error
		}

		return nil
	}); err == nil {
		stats.Records = data
		status = http.StatusOK

		// filter records to their latest state
		summaries := map[string]common.ResHistory{}
		for _, record := range data {
			summaries[record.Hash] = record
		}

		// count stats
		byUser := map[string]common.ResStatCount{}
		for _, rec := range summaries {
			// skip future reservations
			if rec.Start.After(end) {
				continue
			}
			// keep duration calculation wrt the stat window
			thisStart := rec.Start
			if rec.Start.Before(start) {
				thisStart = start
			}
			thisEnd := rec.End
			if rec.End.After(end) {
				thisEnd = end
			}
			if r, ok := byUser[rec.Owner]; ok {
				r.TotalResTime += thisEnd.Sub(thisStart)
				r.ResCount += 1
				r.NodesUsedCount += len(strings.Split(rec.Hosts, ","))
				if rec.Status == HrDeleted {
					r.CancelledEarly += 1
				}
				r.NumExtensions += rec.ExtendCount
				r.Entries = append(r.Entries, rec)
				byUser[rec.Owner] = r

			} else {
				newStats := common.ResStatCount{
					UniqueUsers:    1,
					NodesUsedCount: len(strings.Split(rec.Hosts, ",")),
					ResCount:       1,
					CancelledEarly: 0,
					NumExtensions:  rec.ExtendCount,
					TotalResTime:   thisEnd.Sub(thisStart),
					Entries:        []common.ResHistory{rec},
				}
				if rec.Status == HrDeleted {
					newStats.CancelledEarly += 1
				}
				byUser[rec.Owner] = newStats
			}
		}

		global := common.ResStatCount{
			UniqueUsers:    0,
			NodesUsedCount: 0,
			ResCount:       0,
			CancelledEarly: 0,
			NumExtensions:  0,
			TotalResTime:   time.Minute * 0,
		}
		for _, stats := range byUser {
			global.UniqueUsers += 1
			global.NodesUsedCount += stats.NodesUsedCount
			global.ResCount += stats.ResCount
			global.CancelledEarly += stats.CancelledEarly
			global.NumExtensions += stats.NumExtensions
			global.TotalResTime += stats.TotalResTime
		}
		stats.ByUser = byUser
		stats.Global = global
	}

	return
}
