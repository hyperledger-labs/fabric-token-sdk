/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Resolver contains information about an identity and how to retrieve it.
type Resolver struct {
	Name         string `yaml:"name,omitempty"`
	EnrollmentID string
	Default      bool
	GetIdentity  GetIdentityFunc
	Remote       bool
}

type SigService interface {
	IsMe(driver.Identity) bool
	RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
	RegisterVerifier(identity driver.Identity, v driver.Verifier) error
}

type BinderService interface {
	Bind(longTerm driver.Identity, ephemeral driver.Identity) error
}

type IdentityProvider interface {
	// RegisterAuditInfo binds the passed audit info to the passed identity
	RegisterAuditInfo(identity driver.Identity, info []byte) error

	// GetAuditInfo returns the audit info associated to the passed identity, nil if not found
	GetAuditInfo(identity driver.Identity) ([]byte, error)
}
