// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

const (
	FgUp      = 15  // node is up (white)
	FgDown    = 1   // node is down (red)
	FgPowerNA = 214 // node power status unknown (orange)

	BgUnreserved = 0   // node unreserved (black)
	BgResYes     = 2   // node reserved and writable (green)
	BgResNo      = 5   // node reserved and not writable (magenta)
	BgBlocked    = 3   // node blocked (yellow)
	BgRestricted = 213 // node restricted from user (bright pink)
	BgError      = 75  // node install error (bright cyan)
)

var (
	noColor bool

	// simplePrint indicates that all output should be simplified, easily parsable plain-text with
	// no ANSI color coding.
	simplePrint bool

	cUnreservedUp      = color.S256(FgUp, BgUnreserved)
	cUnreservedDown    = color.S256(FgDown, BgUnreserved).AddOpts(color.OpBold)
	cUnreservedPowerNA = color.S256(FgPowerNA, BgUnreserved).AddOpts(color.OpBold)
	cInstError         = color.S256(FgUp, BgError).AddOpts(color.OpBold)
	cBlockedUp         = color.S256(FgUp, BgBlocked).AddOpts(color.OpBold)
	cRestrictedUp      = color.S256(FgUp, BgRestricted)

	cOwnerRes = color.S256(15, 2)
	cOtherRes = color.S256(15, 5)

	cOK            = color.S256(FgUp, BgUnreserved).AddOpts(color.OpBold)
	cWarning       = color.S256(214)
	cAlert         = color.S256(9).AddOpts(color.OpBold)
	cMotdUrgent    = cAlert
	cMotdNotUrgent = color.S256(3).AddOpts(color.OpBold)

	cFuture      = color.S256(15, 247) // white text, gray background
	cFutureNodes = color.S256(247, BgUnreserved).AddOpts(color.OpItalic)

	hsAvailable = color.S256(2)
	hsReserved  = color.S256(3)
	pUp         = color.S256(10).AddOpts(color.OpBold)
	pDown       = color.S256(9).AddOpts(color.OpBold)
	pUnknown    = color.S256(3).AddOpts(color.OpBold)

	cRespSuccess = color.FgLightGreen
	cRespWarn    = color.FgYellow
	cRespError   = color.S256(15, 9)
	cRespUnknown = color.FgRed

	igorTableStyle = table.Style{
		Name: "IgorTableStyle",
		Box:  table.StyleBoxLight,
		Color: table.ColorOptions{
			Header:       text.Colors{text.FgHiBlue, text.ReverseVideo, text.Bold},
			Row:          text.Colors{text.FgHiWhite, text.BgBlack},
			RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
			Footer:       text.Colors{text.FgHiBlue, text.ReverseVideo, text.Bold},
		},
		Format: table.FormatOptionsDefault,
		HTML:   table.DefaultHTMLOptions,
		Options: table.Options{
			DrawBorder:      false,
			SeparateColumns: true,
			SeparateFooter:  false,
			SeparateHeader:  false,
			SeparateRows:    false},
		Title: table.TitleOptions{
			Colors: append(table.ColorOptionsBlackOnBlueWhite.Header, text.Bold),
		},
	}
)

// sBold use bold style on input text
func sBold(text string) string {
	return color.Bold.Sprint(text)
}

// sItalic use italics style on input text
func sItalic(text string) string {
	return color.OpItalic.Sprint(text)
}

// checkColorLevel turn off terminal color if not supported
func checkColorLevel() {

	if simplePrint || noColor || envNoColor || color.TermColorLevel() == color.LevelNo {
		text.DisableColors()
		color.Disable()
	}
}
