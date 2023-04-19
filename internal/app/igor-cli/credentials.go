// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func reqUserCreds(osUser *user.User) (string, string, error) {

	userIgorDir := filepath.Join(osUser.HomeDir, ".igor")

	if _, err := os.Stat(userIgorDir); errors.Is(err, os.ErrNotExist) {
		if err2 := os.MkdirAll(userIgorDir, 0700); err2 != nil {
			return "", "", err2
		}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("igor username (enter = %s) : ", osUser.Username)
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if len(username) == 0 {
		username = osUser.Username
	}

	if username != "igor-admin" {
		fmt.Print(cli.Client.PasswordLabel + " password: ")
	} else {
		fmt.Print("igor-admin password: ")
	}
	bPswd, err := terminal.ReadPassword(0)
	if err != nil {
		return "", "", err
	}

	fmt.Println("")
	password := string(bPswd)

	return username, password, nil
}

func reqPassChange(name string) (string, string, error) {

	fmt.Printf("(%s) old igor password: ", name)
	bOldPass, err := terminal.ReadPassword(0)
	if err != nil {
		return "", "", err
	}
	fmt.Println("")

	fmt.Printf("(%s) new igor password: ", name)
	bNewPass, err := terminal.ReadPassword(0)
	if err != nil {
		return "", "", err
	}
	fmt.Println("")

	return string(bOldPass), string(bNewPass), nil
}
