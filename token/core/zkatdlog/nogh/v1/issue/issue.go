/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
)

// Prover is the interface for generating zero-knowledge proofs for issue actions.
type Prover interface {
	// Prove generates the zero-knowledge proof of validity.
	Prove() ([]byte, error)
	// RangeProofType returns the type of range proof used by this prover.
	RangeProofType() rp.ProofType
}

// Verifier is the interface for verifying zero-knowledge proofs for issue actions.
type Verifier interface {
	// Verify checks the validity of the zero-knowledge proof for an issue action.
	Verify(proof []byte) error
}

// NewProver returns a new Prover instance based on the public parameters.
// It selects between BulletProof and CSP-based implementations depending on
// whether CSPRangeProofParams is set in the public parameters.
func NewProver(tw []*token.Metadata, tokens []*math.G1, pp *v1.PublicParams) (Prover, error) {
	if pp.CSPRangeProofParams != nil {
		return NewCSPBasedProver(tw, tokens, pp)
	}

	return NewBulletProofProver(tw, tokens, pp)
}

// NewVerifier returns a new Verifier instance based on the public parameters.
// It selects between BulletProof and CSP-based implementations depending on
// whether CSPRangeProofParams is set in the public parameters.
func NewVerifier(tokens []*math.G1, pp *v1.PublicParams) Verifier {
	if pp.CSPRangeProofParams != nil {
		return NewCSPVerifier(tokens, pp)
	}

	return NewBulletProofVerifier(tokens, pp)
}
