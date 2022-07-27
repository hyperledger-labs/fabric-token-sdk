/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

type SigningIdentity interface {
	driver.SigningIdentity
}
