// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"github.com/spf13/cobra"
	"os"
	"path"
	"strings"
	"time"
)

func getLocTime(t time.Time) time.Time {
	return t.In(cli.tzLoc)
}

func openFile(f string) *os.File {
	// get separate path and filename
	fPath, fName := path.Split(f)
	// track working directory
	pwd, err := os.Getwd()
	if err != nil {
		checkClientErr(err)
	}
	// change to the file directory, we need to do
	// this in order for open to record only the
	// file name and not the entire path as metadata
	if err = os.Chdir(fPath); err != nil {
		checkClientErr(err)
	}
	// open the file locally
	r, err := os.Open(fName)
	if err != nil {
		checkClientErr(err)
	}
	// go back to the working directory
	if err = os.Chdir(pwd); err != nil {
		checkClientErr(err)
	}
	return r
}

func validateNoArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

func validateNameArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{"NAME"}, cobra.ShellCompDirectiveNoFileComp
}

func registerFlagArgsFunc(igorCmd *cobra.Command, flagName string, flagArgs []string) error {
	return igorCmd.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return flagArgs, cobra.ShellCompDirectiveNoFileComp
	})
}

// multilineRange takes a node range string and inserts line breaks after the last comma's
// index <= lineWidth, then indents each row after the first. If rangeLine fits
// completely with lineWidth, the method returns the original rangeLine.
func multilineRange(lineWidth int, rangeLine string, prefix string) string {

	getNextLine := func(nl string) (string, string) {
		if len(nl) > lineWidth {
			firstPart := nl[:lineWidth]
			secondPart := nl[lineWidth:]
			fpLastComma := strings.LastIndex(firstPart, ",")
			return firstPart[:fpLastComma+1], firstPart[fpLastComma+1:] + secondPart
		}
		return nl, ""
	}

	lastLine := strings.TrimSpace(rangeLine)
	if len(lastLine) <= lineWidth {
		return lastLine
	}

	lineWithBreaks := ""

	for lastLine != "" {
		if len(lineWithBreaks) > 0 {
			lastLine = strings.Repeat(" ", len(prefix)+1) + lastLine
		}
		var line string
		line, lastLine = getNextLine(lastLine)
		if len(lineWithBreaks) > 0 {
			lineWithBreaks += "\n"
		}
		lineWithBreaks += line
	}

	return lineWithBreaks
}
