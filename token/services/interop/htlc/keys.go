/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
)

const (
	ClaimPreImage = keys.ClaimPreImage
)

// ClaimKey returns the claim key for the passed byte array
func ClaimKey(v []byte) string {
	return ClaimPreImage + base64.StdEncoding.EncodeToString(v)
}
