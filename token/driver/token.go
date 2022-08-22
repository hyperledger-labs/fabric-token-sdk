/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenService interface {
	// DeserializeToken unmarshals the passed output and uses the passed metadata to derive a token and its issuer (if any).
	DeserializeToken(output []byte, outputMetadata []byte) (*token2.Token, view.Identity, error)
}
