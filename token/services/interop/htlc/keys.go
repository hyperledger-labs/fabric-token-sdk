/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/base64"
)

const (
	ClaimPreImage = "htlc.cpi"
	LockHash      = "htlc.lh"
)

// ClaimKey returns the claim key for the passed byte array
func ClaimKey(v []byte) string {
	return ClaimPreImage + base64.StdEncoding.EncodeToString(v)
}

// LockKey returns the lock key for the passed byte array
func LockKey(v []byte) string {
	return LockHash + base64.StdEncoding.EncodeToString(v)
}

// LockValue returns the encoding of the value for a lock key
func LockValue(v []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(v))
}
