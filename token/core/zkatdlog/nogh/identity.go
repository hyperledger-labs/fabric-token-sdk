/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import api2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/api"

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// signing identity
type SigningIdentity interface {
	api2.SigningIdentity
}
