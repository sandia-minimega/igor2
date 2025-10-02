// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"archive/tar"
	"compress/gzip"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"
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

func compressFolderToTarGz(folderPath, tarGzFilePath string) error {
	tarGzFile, err := os.Create(tarGzFilePath)
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(folderPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filePath == folderPath {
			return nil
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filePath)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarWriter, file)
		}
		return err
	})
	return err
}

func validateNoArgs(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

func validateNameArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
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

// multiline breaks the input string into multiple lines up to lineWidth runes each.
// It first trims the input, then for each candidate chunk (of length lineWidth):
//
//  1. If the candidate does not contain any whitespace or comma, it immediately
//     applies the ellipsis condition: it takes lineWidth-2 runes, appends " …",
//     writes that line, and stops further processing.
//
// 2. Otherwise, it scans the candidate for break candidates:
//
//   - Whitespace (tracked by lastSpace)
//
//   - Comma (tracked by lastComma)
//
//   - Hyphen groups: only groups of one or two hyphens are valid (tracked by lastHyphen)
//
//     It then selects the break candidate that occurs furthest to the right.
//     If the break is on a hyphen, the hyphen or hyphen group is included in the output.
//     If it's on whitespace or a comma, the break is applied at that character (trimming trailing spaces for whitespace)
//     and any following whitespace is skipped.
//
// The final output does not include a trailing newline.
func multiline(lineWidth int, line string) string {

	runes := []rune(strings.TrimSpace(line))
	var result strings.Builder
	i := 0

	for i < len(runes) {
		remaining := len(runes) - i
		if remaining <= lineWidth {
			result.WriteString(string(runes[i:]))
			break
		}

		candidate := runes[i : i+lineWidth]

		if strings.IndexFunc(string(candidate), func(r rune) bool {
			return unicode.IsSpace(r) || r == ','
		}) == -1 {
			truncated := string(runes[i:i+lineWidth-2]) + " …"
			result.WriteString(truncated)
			break
		}

		lastSpace := -1
		lastComma := -1
		lastHyphen := -1

		// Scan the candidate for break points.
		for j := 0; j < len(candidate); j++ {
			r := candidate[j]
			// Record whitespace.
			if unicode.IsSpace(r) {
				lastSpace = j
			}
			// Record comma.
			if r == ',' {
				lastComma = j
			}
			// Process hyphens.
			if r == '-' {
				groupStart := j
				groupEnd := j
				for groupEnd < len(candidate) && candidate[groupEnd] == '-' {
					groupEnd++
				}
				groupLen := groupEnd - groupStart
				// Accept groups of 1 or 2 as valid break candidates.
				if groupLen <= 2 {
					if groupEnd-1 > lastHyphen {
						lastHyphen = groupEnd - 1
					}
				}
				// Skip the whole group.
				j = groupEnd - 1
			}
		}

		// If no break candidate was found, apply ellipsis condition.
		// (This situation should rarely occur because the immediate check should have caught
		// candidates with no whitespace or comma.)
		if lastSpace == -1 && lastComma == -1 && lastHyphen == -1 {
			truncated := string(runes[i:i+lineWidth-2]) + " …"
			result.WriteString(truncated)
			break
		}

		// Determine the rightmost break candidate.
		breakIndex := lastSpace
		breakType := "space"
		if lastComma > breakIndex {
			breakIndex = lastComma
			breakType = "comma"
		}
		if lastHyphen > breakIndex {
			breakIndex = lastHyphen
			breakType = "hyphen"
		}

		if breakType == "hyphen" {
			// Include the hyphen (or hyphen group) in the output.
			segment := string(runes[i : i+breakIndex+1])
			result.WriteString(segment)
			result.WriteString("\n")
			// Continue after the hyphen.
			i += breakIndex + 1
		} else if breakType == "comma" {
			// Break at the comma (include it) and then skip it.
			segment := string(runes[i : i+breakIndex+1])
			result.WriteString(segment)
			result.WriteString("\n")
			// Skip any following whitespace.
			i += breakIndex + 1
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
		} else { // breakType == "space"
			// Break at whitespace: trim trailing spaces.
			segment := strings.TrimRight(string(runes[i:i+breakIndex]), " \t")
			result.WriteString(segment)
			result.WriteString("\n")
			// Skip following whitespace.
			i += breakIndex
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
		}
	}

	out := result.String()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out
}

type unit struct {
	s    string // either one printable rune or a whole ANSI code
	ansi bool
}

// multilineNodeList takes a node-range string and inserts line breaks after the
// last comma whose index (in printable runes) is ≤ lineWidth. ANSI codes are
// treated as zero-width and preserved. Active styles at a wrap point are re-applied
// at the start of the next line (after a plain-space indent). Each inserted
// line break ends the line with ESC[0m so styles don’t bleed.
func multilineNodeList(lineWidth int, rangeLine string, prefix string) string {
	// Fast path: if visible length already fits, return as-is.
	if visibleLen(rangeLine) <= lineWidth {
		return rangeLine
	}

	indent := strings.Repeat(" ", len(prefix)+1)

	// Tokenize into units: [ ANSI | single-rune text ]
	src := toUnits(rangeLine)

	var (
		lines []string

		// Current line buffer and accounting
		cur              []unit
		visCount         int // visible runes in cur
		preferredBreakAt = -1
		lastCommaIdx     = -1 // index in cur of the last comma
		lastCommaAtWidth = -1 // lastCommaIdx snapshot when visCount == lineWidth
		activeSGR        []string
	)

	// Helper to append a unit and update state.
	appendUnit := func(u unit) {
		cur = append(cur, u)
		if u.ansi {
			if isReset(u.s) {
				activeSGR = activeSGR[:0]
			} else {
				// Keep a simple list of codes since last reset; re-emit in order.
				activeSGR = append(activeSGR, u.s)
			}
			return
		}
		// Text (single rune)
		visCount++
		if u.s == "," {
			lastCommaIdx = len(cur) - 1
		}
		if visCount == lineWidth {
			preferredBreakAt = len(cur) // right after this unit
			lastCommaAtWidth = lastCommaIdx
		}
	}

	// Main consume loop
	for len(src) > 0 {
		u := src[0]
		src = src[1:]
		appendUnit(u)

		if visCount > lineWidth {
			// Overflow — decide where to break.
			breakAt := preferredBreakAt
			if lastCommaAtWidth >= 0 {
				breakAt = lastCommaAtWidth + 1 // include the comma on this line
			}

			// Emit current line (ensure a reset at the end to prevent bleed).
			line := joinUnits(cur[:breakAt]) + resetSGR
			lines = append(lines, line)

			// Remainder of current buffer stays for the next line.
			remainder := append([]unit(nil), cur[breakAt:]...)

			// Reset line counters and seed next line with indent + active SGRs + remainder.
			cur = cur[:0]
			visCount = 0
			preferredBreakAt = -1
			lastCommaIdx = -1
			lastCommaAtWidth = -1

			// Indent is plain spaces (un-styled).
			if indent != "" {
				for _, r := range indent {
					cur = append(cur, unit{s: string(r), ansi: false})
					visCount++ // indent counts toward width, matching original behavior
				}
			}
			// Re-apply styles active at the break point (after the indent).
			for _, code := range activeSGR {
				cur = append(cur, unit{s: code, ansi: true})
			}
			// Append remainder units and recompute comma/preferred break if needed.
			for _, ru := range remainder {
				appendUnit(ru)
			}
		}
	}

	// Flush final line as-is (no forced reset; preserve original trailing codes).
	if len(cur) > 0 {
		lines = append(lines, joinUnits(cur))
	}

	return strings.Join(lines, "\n")
}

func toUnits(s string) []unit {
	var out []unit
	idxs := ansiRE.FindAllStringIndex(s, -1)
	last := 0
	for _, span := range idxs {
		if span[0] > last {
			// Split text into single-rune units
			for _, r := range s[last:span[0]] {
				out = append(out, unit{s: string(r), ansi: false})
			}
		}
		out = append(out, unit{s: s[span[0]:span[1]], ansi: true})
		last = span[1]
	}
	if last < len(s) {
		for _, r := range s[last:] {
			out = append(out, unit{s: string(r), ansi: false})
		}
	}
	return out
}

func joinUnits(us []unit) string {
	var b strings.Builder
	for _, u := range us {
		b.WriteString(u.s)
	}
	return b.String()
}

func visibleLen(s string) int {
	plain := ansiRE.ReplaceAllString(s, "")
	// Count runes
	return len([]rune(plain))
}

func isReset(code string) bool {
	return code == resetSGR || resetRE.MatchString(code)
}
