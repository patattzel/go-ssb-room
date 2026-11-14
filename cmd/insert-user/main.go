// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// insert-user is a utility to create a new member and fallback password for them
package main

import (
	"bytes"
	"context"
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
	)

	flag.StringVar(&repoPath, "repo", filepath.Join(u.HomeDir, ".ssb-go-room"), "[optional] where the locally stored files of the room are located")
	flag.BoolVar(&resetPassword, "reset-password", false, "set a new password for an existing member")
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

	_, err = os.Stat(filepath.Join(repoPath, "roomdb"))
	check(err)

	// we require one more argument which is not a flag.
	if len(flag.Args()) != 1 {
		cliMissingArguments("please provide a public key")
	}

	pubKey, err := refs.ParseFeedRef(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid ssb public-key reference:", err)
		os.Exit(1)
	}

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

func cliMissingArguments(message string) {
	executable := strings.TrimPrefix(os.Args[0], "./")
	fmt.Fprintf(os.Stderr, "%s: %s\nusage:%s <optional flags> <@base64-encoded-public-key=.ed25519>\n", executable, message, executable)
	flag.Usage()
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
