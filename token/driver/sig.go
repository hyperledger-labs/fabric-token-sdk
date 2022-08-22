/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// SigningIdentity models a signing identity
type SigningIdentity interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(raw []byte) ([]byte, error)

	// Verify verifies a signature over a message
	Verify(message, sigma []byte) error

	// Serialize serializes the signing identity
	Serialize() ([]byte, error)
}

// Verifier is an interface which wraps the Verify method.
type Verifier interface {
	// Verify verifies the signature over the message bytes and returns nil if the signature is valid and an error otherwise.
	Verify(message, sigma []byte) error
}

// Signer is an interface which wraps the Sign method.
type Signer interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(message []byte) ([]byte, error)
}
