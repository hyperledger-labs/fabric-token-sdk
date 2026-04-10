/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
)

type Prover interface {
	Prove() ([]byte, error)
	RangeProofType() rp.ProofType
}

// NewProver returns a new Prover instance.
func NewProver(inputWitness, outputWitness []*token.Metadata, inputs, outputs []*math.G1, pp *v1.PublicParams) (Prover, error) {
	if pp.CSPRangeProofParams != nil {
		return NewCSPBasedProver(inputWitness, outputWitness, inputs, outputs, pp)
	}

	return NewBulletProofProver(inputWitness, outputWitness, inputs, outputs, pp)
}

type Verifier interface {
	Verify(proofRaw []byte) error
}

func NewVerifier(inputs, outputs []*math.G1, pp *v1.PublicParams, proofType rp.ProofType) Verifier {
	if proofType == rp.RangeProofType {
		return NewBulletProofVerifier(inputs, outputs, pp)
	}

	return NewCSPVerifier(inputs, outputs, pp)
}
