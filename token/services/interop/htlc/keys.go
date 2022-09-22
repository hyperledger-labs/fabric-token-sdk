/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
)

func ClaimKey(v []byte) string {
	return keys.ClaimPreImage + base64.StdEncoding.EncodeToString(v)
}
