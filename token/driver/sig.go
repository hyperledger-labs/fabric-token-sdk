/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// FullIdentity defines an identity that can both sign messages and verify signatures.
type FullIdentity interface {
	SigningIdentity
	Verifier
}

// SigningIdentity represents an identity capable of generating cryptographic signatures.
type SigningIdentity interface {
	// Sign signs the provided message bytes and returns the resulting signature.
	Sign(raw []byte) ([]byte, error)

	// Serialize converts the signing identity into its byte representation.
	Serialize() ([]byte, error)
}

//go:generate counterfeiter -o mock/verifier.go -fake-name Verifier . Verifier

// Verifier defines the interface for verifying cryptographic signatures.
type Verifier interface {
	// Verify checks the signature against the provided message and returns nil if valid.
	Verify(message, sigma []byte) error
}

//go:generate counterfeiter -o mock/signer.go -fake-name Signer . Signer

// Signer defines the interface for generating cryptographic signatures.
type Signer interface {
	// Sign signs the provided message and returns the resulting signature.
	Sign(message []byte) ([]byte, error)
}
