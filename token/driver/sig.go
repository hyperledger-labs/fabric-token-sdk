/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type FullIdentity interface {
	SigningIdentity
	Verifier
}

// SigningIdentity models a signing identity
type SigningIdentity interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(raw []byte) ([]byte, error)

	// Serialize serializes the signing identity
	Serialize() ([]byte, error)
}

//go:generate counterfeiter -o mock/verifier.go -fake-name Verifier . Verifier

// Verifier is an interface which wraps the Verify method.
type Verifier interface {
	// Verify verifies the signature over the message bytes and returns nil if the signature is valid and an error otherwise.
	Verify(message, sigma []byte) error
}

//go:generate counterfeiter -o mock/signer.go -fake-name Signer . Signer

// Signer is an interface which wraps the Sign method.
type Signer interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(message []byte) ([]byte, error)
}

// VerifierDeserializer is the interface for verifiers' deserializer.
// A verifier checks the validity of a signature against the identity associated with the verifier
type VerifierDeserializer interface {
	DeserializeVerifier(id Identity) (Verifier, error)
}
