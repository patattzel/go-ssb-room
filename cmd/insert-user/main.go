// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// insert-user is a utility to create a new member and fallback password for them
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/repo"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/roomdb/sqlite"
)

func main() {
	u, err := user.Current()
	check(err)

	var (
		role          roomdb.Role = roomdb.RoleAdmin
		repoPath      string
		resetPassword bool
		promptID      bool
	)

	flag.StringVar(&repoPath, "repo", filepath.Join(u.HomeDir, ".ssb-go-room"), "[optional] where the locally stored files of the room are located")
	flag.BoolVar(&resetPassword, "reset-password", false, "set a new password for an existing member")
	flag.BoolVar(&promptID, "prompt-id", false, "prompt for the public SSB ID in the terminal")
	flag.Func("role", "[optional] which role the new member should have (values: mod[erator], admin, or member. default is admin)", func(val string) error {
		switch strings.ToLower(val) {
		case "admin":
			role = roomdb.RoleAdmin
		case "mod":
			fallthrough
		case "moderator":
			role = roomdb.RoleModerator
		case "member":
			role = roomdb.RoleMember
		default:
			return fmt.Errorf("unknown member role: %q", val)
		}

		return nil
	})
	flag.Parse()

	parseID := parseAdminID
	idPrompt := "Public SSB ID from your Manyverse profile (@….ed25519): "
	if resetPassword || role != roomdb.RoleAdmin {
		parseID = func(value string) (refs.FeedRef, error) {
			return refs.ParseFeedRef(strings.TrimSpace(value))
		}
		idPrompt = "Public SSB ID: "
	}

	var pubKey refs.FeedRef
	switch {
	case promptID && flag.NArg() == 0:
		input := bufio.NewScanner(os.Stdin)
		for {
			fmt.Fprint(os.Stderr, idPrompt)
			if !input.Scan() {
				check(input.Err())
				check(errors.New("no SSB ID entered"))
			}
			pubKey, err = parseID(input.Text())
			if err == nil {
				break
			}
			fmt.Fprintln(os.Stderr, err)
		}
	case !promptID && flag.NArg() == 1:
		pubKey, err = parseID(flag.Arg(0))
		check(err)
	default:
		cliMissingArguments("provide one public SSB ID or use -prompt-id")
	}

	_, err = os.Stat(filepath.Join(repoPath, "roomdb"))
	check(err)

	r := repo.New(repoPath)
	db, err := sqlite.Open(r)
	check(err)
	defer db.Close()

	fmt.Fprint(os.Stderr, "Choose a web password (12–72 bytes): ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	check(err)

	fmt.Fprint(os.Stderr, "Repeat password: ")
	bytePasswordRepeat, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	check(err)

	if !bytes.Equal(bytePassword, bytePasswordRepeat) {
		fmt.Fprintln(os.Stderr, "Passwords didn't match")
		os.Exit(1)
	}
	if len(bytePassword) < 12 || len(bytePassword) > 72 {
		check(fmt.Errorf("password must be 12–72 bytes"))
	}

	ctx := context.Background()
	if resetPassword {
		member, err := db.Members.GetByFeed(ctx, pubKey)
		check(err)
		check(db.AuthFallback.SetPassword(ctx, member.ID, string(bytePassword)))
		fmt.Fprintln(os.Stderr, "Updated the member's web password.")
		return
	}
	mid, err := db.AddMemberWithPassword(ctx, pubKey, role, string(bytePassword))
	check(err)

	fmt.Fprintf(os.Stderr, "Created member (%s) with ID %d\n", role, mid)
}

func parseAdminID(value string) (refs.FeedRef, error) {
	value = strings.TrimSpace(value)
	const formatHelp = "Copy the public SSB ID from your Manyverse profile. It starts with @ and ends with .ed25519."
	if strings.HasPrefix(strings.ToLower(value), "ssb:") {
		return refs.FeedRef{}, errors.New("This is an SSB URI. " + formatHelp)
	}
	if strings.HasPrefix(value, "@") && strings.HasSuffix(value, ".ed25519") {
		id, err := refs.ParseFeedRef(value)
		if err == nil && id.Algo() == refs.RefAlgoFeedSSB1 && id.Sigil() == value {
			return id, nil
		}
	}
	return refs.FeedRef{}, errors.New("Invalid SSB ID. " + formatHelp)
}

func cliMissingArguments(message string) {
	executable := strings.TrimPrefix(os.Args[0], "./")
	fmt.Fprintf(os.Stderr, "%s: %s\nusage: %s [options] <@public-key=.ed25519>\n", executable, message, executable)
	flag.Usage()
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
