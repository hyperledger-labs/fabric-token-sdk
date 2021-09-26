/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type SigningIdentity interface {
	Sign(raw []byte) ([]byte, error)

	Verify(message, sigma []byte) error

	Serialize() ([]byte, error)
}

type Verifier interface {
	Verify(message, sigma []byte) error
}

// Signer is an interface which wraps the Sign method.
type Signer interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(message []byte) ([]byte, error)
}

type SignerService interface {
	// RegisterSigner associated the passed signer and verifier to the passed identity
	RegisterSigner(identity view.Identity, signer Signer, verifier Verifier) error
}
