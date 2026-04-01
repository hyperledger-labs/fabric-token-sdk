/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type (
	// IdentityType identifies the type of identity.
	// It is an alias for tdriver.IdentityType and is used by deserializers to choose the correct
	// decoding logic for different identity representations.
	IdentityType = tdriver.IdentityType

	IdentityTypeString = string

	Identity = tdriver.Identity
)
