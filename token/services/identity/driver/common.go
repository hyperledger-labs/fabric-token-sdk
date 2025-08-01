/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type SigService interface {
	IsMe(context.Context, driver.Identity) bool
	RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
	RegisterVerifier(ctx context.Context, identity driver.Identity, v driver.Verifier) error
}

type NetworkBinderService interface {
	Bind(ctx context.Context, longTerm driver.Identity, ephemeral driver.Identity) error
}

type BinderService interface {
	Bind(ctx context.Context, longTerm driver.Identity, ephemeral driver.Identity, copyAll bool) error
}

type IdentityProvider interface {
	// RegisterAuditInfo binds the passed audit info to the passed identity
	RegisterAuditInfo(ctx context.Context, identity driver.Identity, info []byte) error

	// GetAuditInfo returns the audit info associated to the passed identity, nil if not found
	GetAuditInfo(ctx context.Context, identity driver.Identity) ([]byte, error)
}
