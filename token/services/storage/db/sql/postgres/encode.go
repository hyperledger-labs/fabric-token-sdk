/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"encoding/hex"
	"strings"
)

func identity(a string) (string, error) { return a, nil }

// decodeBYTEA decodes a postgres response of type BYTEA
func decodeBYTEA(s string) (string, error) {
	// we only decode if we have indeed a BYTEA (returned as hex)
	if !strings.HasPrefix(s, "\\x") {
		return s, nil
	}

	b, err := hex.DecodeString(s[2:])
	if err != nil {
		return "", err
	}

	return string(b), err
}
