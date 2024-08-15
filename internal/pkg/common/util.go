// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// 2023-03-01T01:04:10Z

const (
	oneDay                 = time.Hour * 24
	DateTimeLogFormat      = "2006-01-02T15:04:05.000"
	DateTimeFilenameFormat = "2006-01-02T15-04-05.000"
	DateTimePublicFormat   = "2006-01-02T15:04:05-07:00"
	DateTimeCompactFormat  = "Jan-2-06T15:04"
	DateTimeLongFormat     = "Jan-02-2006 15:04:05"
	DateTimeServerFormat   = "Jan 2 2006 15:04:05 MST(-07:00)"
	DateTimeEmailFormat    = "January 2, 2006 - 3:04 PM MST"

	IgorRefreshHeader = "X-Igor-Refresh"

	Authorization = "Authorization"
	ContentLength = "Content-Length"
	ContentType   = "Content-Type"
	Referer       = "Referer"
	UserAgent     = "User-Agent"
	Origin        = "Origin"
	XForwardedFor = "X-Forwarded-For"

	// MIME-types

	MAppJson   = "application/json"
	MFormData  = "multipart/form-data"
	MTextPlain = "text/plain"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var smallLetters = []rune("abcdefghijklmnopqrstuvwxyz")

// RandSeq generates a random sequence of characters from the above letters of given length n
// Should NOT be used for passwords/security related needs
func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = smallLetters[rand.Intn(len(smallLetters))]
	}
	return string(b)
}

func WriteFile(path string, content string, mode os.FileMode) error {
	if file, err := os.Create(path); err != nil {
		return fmt.Errorf("failed to create %v -- %v", path, err)
	} else {
		_ = file.Chmod(mode)
		_, err = file.WriteString(content)
		if err != nil {
			return fmt.Errorf("failed to write %v -- %v", path, err)
		}
		_ = file.Close()
	}
	return nil
}

// ParseDuration parses a duration, supporting a 'd' suffix in addition to
// those supported by time.ParseDuration. Rounds duration to minute. It
// will remove spaces so that durations produced from FormatDuration with
// alignment columns can be understood.
func ParseDuration(s string) (time.Duration, error) {
	// unitless integer is assumed to be in minutes

	s = strings.ReplaceAll(s, " ", "")

	if v, err := strconv.Atoi(s); err == nil {
		return time.Duration(v) * time.Minute, nil
	}

	var d time.Duration

	index := strings.Index(s, "d")
	if index > 0 {
		days, err := strconv.Atoi(s[:index])
		if err != nil {
			return -1, err
		}
		d = time.Duration(days*24) * time.Hour
	}

	if index+1 < len(s) {
		v, err := time.ParseDuration(s[index+1:])
		if err != nil {
			return -1, err
		}

		d += v
	}

	return d.Round(time.Minute), nil
}

// FormatDuration will output a duration value as a string by building up fields to
// display including days/hours/minutes (Ex: "3d21h9m"). The alignColumns
// param will put each field in a separated, vertical left-aligned column.
func FormatDuration(d time.Duration, alignColumns bool) string {

	days := d / oneDay
	d -= days * oneDay

	hours := d / time.Hour
	d -= hours * time.Hour

	minutes := d / time.Minute
	d -= minutes * time.Minute

	var final string

	if minutes != 0 {
		final += fmt.Sprintf(" %2dm", minutes)
	} else {
		final += "  0m"
	}

	if hours != 0 {
		final = fmt.Sprintf(" %2dh", hours) + final
	} else {
		if days != 0 {
			final = "  0h" + final
		}
	}

	if days != 0 {
		final = fmt.Sprintf("%dd", days) + final
	}

	if d != 0 {
		return "< 1m"
	} else if final == "" {
		return "---"
	}

	if !alignColumns {
		return strings.ReplaceAll(final, " ", "")
	}

	return strings.TrimSpace(final)
}

// ParseTimeFormat checks that the input string matches any of the expected datetime
// formats igor recognizes.
func ParseTimeFormat(t string) (timeVal time.Time, err error) {

	if timeVal, err = time.ParseInLocation(DateTimeCompactFormat, t, time.Local); err != nil {
		err = fmt.Errorf("unrecognized datetime format: %v", err)
	}
	return
}
