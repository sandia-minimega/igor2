// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import "time"

// ScheduleTimer is a repeating timer that will re-fire at a duration specified by the function calcDur when
// the reset method is called. The calcDir function can either specify a regular interval or how much time
// is left based on some repeating wall clock value like noon or 5:30 AM. It should never return a duration based
// on a fixed datetime that only happens once.
type ScheduleTimer struct {
	t       *time.Timer
	calcDur func(time.Duration) time.Duration
	off     time.Duration
}

func (st *ScheduleTimer) reset() {
	st.t.Reset(st.calcDur(st.off))
}

// NewScheduleTimer creates a new ScheduleTimer that fires at the top of every minute plus the offset provided
// by input.  The offset must be at least one minute, otherwise the timer with loop infinitely.
//
// Note that if any offset includes seconds it will cause the timer to fire at the second mark of that wall clock
// minute. Thus if the offset is 5 minutes and 30 seconds, the timer will fire every 5 minutes at that minute's
// 30-second mark, i.e., 8:02:30, 8:07:30, 8:12:30, ...
func NewScheduleTimer(offset time.Duration) ScheduleTimer {
	dur := getDurationToClockTime(offset)
	if dur >= 5*time.Second {
		// this condition only affects startup. If the first timer is more than 5 seconds away, just do it now, and
		// it will fire at the next assigned interval (which could be as close as 5 seconds)
		return ScheduleTimer{time.NewTimer(10 * time.Millisecond), getDurationToClockTime, offset}
	}
	// otherwise, just fire at the next assigned interval (waiting up to 5 seconds to do so).
	return ScheduleTimer{time.NewTimer(dur), getDurationToClockTime, offset}
}

// getDurationToClockTime uses the current local datetime floored to the minute (i.e., 10:27:45 -> 10:27) as the basis
// for adding offsets to derive a new time interval until a certain point in the future. Offsets can be any
// number of time values (including seconds) to add to the base time. The total amount of the offset must be at least
// one minute if used in a ScheduleTimer, otherwise the timer with loop infinitely.
func getDurationToClockTime(offset time.Duration) time.Duration {
	now := time.Now()
	nextInterval :=
		time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.Local).Add(offset)
	return time.Until(nextInterval)
}
