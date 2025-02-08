/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// SameType shows that issued tokens contains Pedersen commitments to (type, value)
// SameType also shows that all the issued tokens contain the same type
type SameType struct {
	// Proof of type
	Type *math.Zr
	// Proof of randomness used to compute the commitment to type and value in the issued tokens
	// i^th proof is for the randomness  used to compute the i^th token
	BlindingFactor *math.Zr
	// only when the type is not hidden
	TypeInTheClear string
	// Challenge computed using the Fiat-Shamir Heuristic
	Challenge *math.Zr
	// CommitmentToType is a commitment to the type being issued
	CommitmentToType *math.G1
}

// Serialize marshals SameType proof
func (stp *SameType) Serialize() ([]byte, error) {
	return json.Marshal(stp)
}

// Deserialize un-marshals SameType proof
func (stp *SameType) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, stp)
}

// SameTypeRandomness is the randomness used to generate
// the well-formedness proof
type SameTypeRandomness struct {
	blindingFactor *math.Zr
	tokenType      *math.Zr
}

// SameTypeProver contains information that allows an Issuer to prove that
// issued tokens are have the same type
type SameTypeProver struct {
	PedParams []*math.G1
	Curve     *math.Curve
	// tokenType is the type of the tokens to be issued
	tokenType token2.Type
	// blindingFactor is the blinding factor in the CommitmentToType
	blindingFactor *math.Zr
	// CommitmentToType is a commitment to tokenType using blindingFactor
	CommitmentToType *math.G1
	// randomness is the randomness during the proof generation
	randomness *SameTypeRandomness
	// commitment is the commitment to the randomness used to generate the proof
	commitment *math.G1
}

// NewSameTypeProver returns a SameTypeProver for the passed parameters
func NewSameTypeProver(ttype token2.Type, bf *math.Zr, com *math.G1, pp []*math.G1, c *math.Curve) *SameTypeProver {

	return &SameTypeProver{
		tokenType:        ttype,
		blindingFactor:   bf,
		CommitmentToType: com,
		PedParams:        pp,
		Curve:            c,
	}
}

// Prove returns a SameType proof
func (p *SameTypeProver) Prove() (*SameType, error) {
	tokenType := p.Curve.HashToZr([]byte(p.tokenType))

	// compute commitments used in the Schnorr proof
	err := p.computeCommitment()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't prove type during the issue")
	}
	array := common.GetG1Array([]*math.G1{p.CommitmentToType, p.commitment})
	var toHash []byte
	toHash, err = array.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't prove type during the issue")
	}
	chal := p.Curve.HashToZr(toHash)
	proof := &SameType{
		CommitmentToType: p.CommitmentToType,
		Challenge:        chal,
	}
	proof.Type = p.Curve.ModMul(chal, tokenType, p.Curve.GroupOrder)
	proof.Type = p.Curve.ModAdd(proof.Type, p.randomness.tokenType, p.Curve.GroupOrder)

	proof.BlindingFactor = p.Curve.ModMul(chal, p.blindingFactor, p.Curve.GroupOrder)
	proof.BlindingFactor = p.Curve.ModAdd(proof.BlindingFactor, p.randomness.blindingFactor, p.Curve.GroupOrder)
	return proof, nil

}

// computeCommitment compute the commitmentsto the randomness used in the same type proof
func (p *SameTypeProver) computeCommitment() error {
	// get random number generator
	rand, err := p.Curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	// randomness for proof
	p.randomness = &SameTypeRandomness{}
	p.randomness.tokenType = p.Curve.NewRandomZr(rand)
	p.randomness.blindingFactor = p.Curve.NewRandomZr(rand)

	// compute commitment
	p.commitment = p.PedParams[0].Mul(p.randomness.tokenType)
	p.commitment.Add(p.PedParams[2].Mul(p.randomness.blindingFactor))

	return nil
}

// SameTypeVerifier checks the validity of SameType proof
type SameTypeVerifier struct {
	PedParams []*math.G1
	Curve     *math.Curve
	Tokens    []*math.G1
}

// NewSameTypeVerifier returns a SameTypeVerifier corresponding to the passed parameters
func NewSameTypeVerifier(tokens []*math.G1, pp []*math.G1, c *math.Curve) *SameTypeVerifier {
	return &SameTypeVerifier{
		Tokens:    tokens,
		PedParams: pp,
		Curve:     c,
	}
}

// Verify returns an error if the serialized proof is an invalid SameType proof
func (v *SameTypeVerifier) Verify(proof *SameType) error {
	// recompute commitments used in ZK proofs
	com := v.PedParams[0].Mul(proof.Type)
	com.Add(v.PedParams[2].Mul(proof.BlindingFactor))
	com.Sub(proof.CommitmentToType.Mul(proof.Challenge))

	// recompute challenge and check proof validity
	raw, err := common.GetG1Array([]*math.G1{proof.CommitmentToType, com}).Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to verify same type proof")
	}

	if !v.Curve.HashToZr(raw).Equals(proof.Challenge) {
		return errors.Errorf("invalid same type proof")
	}
	return nil
}
