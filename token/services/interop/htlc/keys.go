/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/hex"
)

const (
	ClaimPreImage = "cpi"
)

// ClaimKey returns the claim key for the passed byte array
func ClaimKey(v []byte) string {
	return ClaimPreImage + hex.EncodeToString(v)
}
