// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBadPasswordCheck(t *testing.T) {

	badPasswords := []string{
		"&fhso",             // too short
		"*fdhs8adGBDiodcbd", // too long
		" pass$ word1",      // whitespace in middle
		"3$pãssword",        // diacriticals
		"1password",         // no special
		"pas$word",          // no number
		"436^&#0384",        // no letter
	}

	for _, p := range badPasswords {
		err := checkLocalPasswordRules(p)
		fmt.Println(err)
		assert.NotNil(t, err, "Bad password %s should have failed but passed", p)
	}
}

func TestGoodPasswordCheck(t *testing.T) {

	goodPasswords := []string{
		"*fdhs8`adGBD",  // rare symbol
		"*fdhs8adGBD\"", // escaped quotes
	}

	for _, p := range goodPasswords {
		err := checkLocalPasswordRules(p)
		assert.Nil(t, err, "Password %s should have passed but failed", p)
	}
}

func TestBadEmailCheck(t *testing.T) {

	badEmails := []string{
		"fhsogmail.com",            // no @ symbol
		"Takeshi.Kovacs@gmail.co1", // no numbers in TLD
		"gfgd @agency.gov",         // non-legal characters
		"me..too@gmail.com",        // adjacent periods
	}

	for _, email := range badEmails {
		err := checkEmailRules(email)
		fmt.Println(err)
		assert.NotNil(t, err, "Bad email %s should have failed but passed", email)
	}
}

func TestGoodEmailCheck(t *testing.T) {

	goodEmails := []string{
		"tester@level1.level2.level3.gov",
	}

	for _, email := range goodEmails {
		err := checkEmailRules(email)
		assert.Nil(t, err, "Email %s should have passed but failed", email)
	}
}
