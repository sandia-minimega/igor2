// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/gookit/color"

	"igor2/internal/pkg/common"
)

const (
	respPrefix = "igor: "
	adminOnly  = "[admin-only]"
)

// printRespSimple prints the message portion of ResponseBody to
// STDOUT with color based on the status field.
func printRespSimple(rb common.ResponseBody) {

	checkColorLevel()

	msg := respPrefix + strings.TrimSpace(rb.GetMessage())
	if len(msg) == len(respPrefix) {
		if rb.IsSuccess() {
			msg += fmt.Sprint("success")
		} else if rb.IsFail() {
			msg += fmt.Sprint("unspecified failure")
		} else if rb.IsError() {
			msg += fmt.Sprint("unspecified error")
		}
	}

	var final string
	if rb.IsSuccess() {
		final = cRespSuccess.Sprint(msg)
	} else if rb.IsFail() {
		final = cRespWarn.Sprint(msg)
	} else if rb.IsError() {
		final = cRespError.Sprint(msg)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "%sunrecognized status - %s\n", respPrefix, cRespUnknown.Sprint(rb.GetMessage()))
		os.Exit(1)
	}

	fmt.Println(final)
	os.Exit(0)
}

// printSimple prints out non-error igor responses that originate in the cli or
// when the server response needs more context.
func printSimple(msg string, mType color.Color) {
	checkColorLevel()
	final := mType.Sprintf("%s%v", respPrefix, msg)
	fmt.Println(final)
	os.Exit(0)
}

// checkClientErr is used for handling errors that originate in the cli. It will
// print and exit with code 1 if the error is not nil.
func checkClientErr(err error) {
	if err != nil {
		checkColorLevel()
		errMsg := color.FgLightRed.Sprintf("%s%v", respPrefix, err)
		fmt.Fprintln(os.Stderr, errMsg)
		os.Exit(1)
	}
}

func checkAndSetColorLevel(rb common.ResponseBody) {

	checkColorLevel()

	if checkRespFailure(rb) {
		printRespSimple(rb)
		os.Exit(1)
	}
}

var adminOnlyBanner = sBold("  --- admin-only command ---")
var requiredArgs = sBold("REQUIRED ARGS:")
var requiredFlags = sBold("REQUIRED FLAGS:")
var optionalFlags = sBold("OPTIONAL FLAGS:")
var notesOnUsage = sBold("NOTES ON USAGE:")
var descFlagText = `Use the --desc flag to set a description should one be desired. This is a text
field up to 256 characters and enclosed in quotes, ex: "A simple description."
Descriptions are visible to all users.`
