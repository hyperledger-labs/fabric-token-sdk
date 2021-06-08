/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package api

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenService interface {
	// DeserializeToken returns the token and its issuer (if any).
	DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, view.Identity, error)
}
