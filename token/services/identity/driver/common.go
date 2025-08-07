/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type NetworkBinderService interface {
	Bind(ctx context.Context, longTerm driver.Identity, ephemeral driver.Identity) error
}

type IdentityProvider interface {
	IsMe(context.Context, driver.Identity) bool

	// Bind an ephemeral identity to another identity
	Bind(ctx context.Context, longTerm driver.Identity, ephemeralIdentities ...driver.Identity) error

	// RegisterIdentityDescriptor register the passed identity descriptor with an alias
	RegisterIdentityDescriptor(ctx context.Context, identityDescriptor *IdentityDescriptor, alias driver.Identity) error
}
