// SPDX-FileCopyrightText: 2026 The go-ssb-room Cloudron packaging contributors
//
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseAdminID(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	valid := "@" + key + ".ed25519"
	for _, input := range []string{valid, "  " + valid + "\r\n"} {
		id, err := parseAdminID(input)
		if err != nil || id.Sigil() != valid {
			t.Fatalf("parseAdminID(%q) = %v, %v, want %s", input, id, err, valid)
		}
	}

	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{"profile URI", "ssb:feed/ed25519/" + key, "This is an SSB URI"},
		{"missing sigil", key + ".ed25519", "Invalid SSB ID"},
		{"wrong algorithm", "@" + key + ".gabbygrove-v1", "Invalid SSB ID"},
		{"short key", "@AAAA.ed25519", "Invalid SSB ID"},
		{"invalid base64", "@!" + key[1:] + ".ed25519", "Invalid SSB ID"},
		{"noncanonical base64", "@" + strings.TrimSuffix(key, "A=") + "B=.ed25519", "Invalid SSB ID"},
		{"additional suffix", valid + ".ed25519", "Invalid SSB ID"},
		{"empty", "", "Invalid SSB ID"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseAdminID(test.input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parseAdminID(%q) error = %v, want %q", test.input, err, test.want)
			}
		})
	}
}
