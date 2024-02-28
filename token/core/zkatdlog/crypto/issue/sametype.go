/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// SameType shows that a bridge commitment is of the form G_0^typeH^r
type SameType struct {
	// Proof of type
	Type *math.Zr
	// Proof of randomness used to compute the commitment to type and value in the issued tokens
	// i^th proof is for the randomness  used to compute the i^th token
	BlindingFactor *math.Zr
	// only when issue is not anonymous
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
	// Anonymous indicates if the issuance is anonymous, and therefore,  the type is hidden
	Anonymous bool
	// tokenType is the type of the tokens to be issued
	tokenType string
	// blindingFactor is the blinding factor in the CommitmentToType
	blindingFactor *math.Zr
	// CommitmentToType is a commitment to tokenType using blindingFactor
	CommitmentToType *math.G1
	// randomness is the randomness during the proof generation
	randomness *SameTypeRandomness
	// commitment is the commitment to the randomness used to generate the proof
	commitment         *math.G1
	PedersenGenerators []*math.G1
	Curve              *math.Curve
}

// NewSameTypeProver returns a SameTypeProver for the passed parameters
func NewSameTypeProver(ttype string, bf *math.Zr, com *math.G1, anonymous bool, pp []*math.G1, c *math.Curve) *SameTypeProver {

	return &SameTypeProver{
		tokenType:          ttype,
		blindingFactor:     bf,
		CommitmentToType:   com,
		Anonymous:          anonymous,
		PedersenGenerators: pp,
		Curve:              c,
	}
}

// Prove returns a SameType proof
func (p *SameTypeProver) Prove() (*SameType, error) {
	tokenType := p.Curve.HashToZr([]byte(p.tokenType))
	if p.Anonymous {
		// the type of the token is hidden
		// prove that commitToType is of the form G_0^typeH^r
		// compute commitments used in the Schnorr proof
		err := p.computeCommitment()
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't prove type during the issue")
		}
		array := common.GetG1Array([]*math.G1{p.CommitmentToType, p.commitment}, p.PedersenGenerators)
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
	proof := &SameType{
		CommitmentToType: p.CommitmentToType,
		TypeInTheClear:   p.tokenType,
	}
	return proof, nil
}

// computeCommitment compute the commitments to the randomness used in the same type proof
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
	p.commitment = p.PedersenGenerators[0].Mul(p.randomness.tokenType)
	p.commitment.Add(p.PedersenGenerators[2].Mul(p.randomness.blindingFactor))

	return nil
}

// SameTypeVerifier checks the validity of SameType proof
type SameTypeVerifier struct {
	PedersenGenerators []*math.G1
	Curve              *math.Curve
	// Tokens to be issued
	Tokens []*math.G1
	// anonymous indicates if the issuance is anonymous
	anonymous bool
}

// NewSameTypeVerifier returns a SameTypeVerifier corresponding to the passed parameters
func NewSameTypeVerifier(tokens []*math.G1, anonymous bool, pp []*math.G1, c *math.Curve) *SameTypeVerifier {
	return &SameTypeVerifier{
		Tokens:             tokens,
		anonymous:          anonymous,
		PedersenGenerators: pp,
		Curve:              c,
	}
}

// Verify returns an error if the serialized proof is an invalid SameType proof
func (v *SameTypeVerifier) Verify(proof *SameType) error {
	// recompute commitments used in ZK proofs
	if v.anonymous {
		// recompute challenge and check proof validity
		com := v.PedersenGenerators[0].Mul(proof.Type)
		com.Add(v.PedersenGenerators[2].Mul(proof.BlindingFactor))
		com.Sub(proof.CommitmentToType.Mul(proof.Challenge))

		// recompute challenge
		raw, err := common.GetG1Array([]*math.G1{proof.CommitmentToType, com}, v.PedersenGenerators).Bytes()
		if err != nil {
			return errors.Wrapf(err, "failed to verify same type proof")
		}

		if !v.Curve.HashToZr(raw).Equals(proof.Challenge) {
			return errors.Errorf("invalid same type proof")
		}
		return nil
	}
	if !proof.CommitmentToType.Equals(v.PedersenGenerators[0].Mul(v.Curve.HashToZr([]byte(proof.TypeInTheClear)))) {
		return errors.Errorf("invalid type in issue")
	}
	return nil
}
