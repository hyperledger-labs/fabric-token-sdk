/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"sync"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/asn1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
)

// Proof is a zero-knowledge proof that shows that a Transfer Action is valid.
// It ensures that:
// 1. Inputs and outputs have the same total value.
// 2. Inputs and outputs have the same token type.
// 3. Output values are within the authorized range (to prevent overflows).
type Proof struct {
	// TypeAndSum is a proof that inputs and outputs have the same total value and token type.
	TypeAndSum *TypeAndSumProof
	// RangeCorrectness is a proof that the outputs have values in the authorized range.
	RangeCorrectness *rp.RangeCorrectness
}

// Serialize marshals the Proof to bytes.
func (p *Proof) Serialize() ([]byte, error) {
	return asn1.Marshal[asn1.Serializer](p.TypeAndSum, p.RangeCorrectness)
}

// Deserialize unmarshals the Proof from bytes.
func (p *Proof) Deserialize(bytes []byte) error {
	p.TypeAndSum = &TypeAndSumProof{}
	p.RangeCorrectness = &rp.RangeCorrectness{}

	return asn1.Unmarshal[asn1.Serializer](bytes, p.TypeAndSum, p.RangeCorrectness)
}

// Validate ensures the proof components are present and well-formed.
func (p *Proof) Validate(curve math.CurveID) error {
	if p.TypeAndSum == nil {
		return ErrMissingTypeAndSumProof
	}
	if err := p.TypeAndSum.Validate(curve); err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}
	if p.RangeCorrectness == nil {
		return nil
	}
	err := p.RangeCorrectness.Validate(curve)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}

	return nil
}

// Verifier verifies if a Transfer Action is valid.
type Verifier struct {
	PP               *v1.PublicParams
	TypeAndSum       *TypeAndSumVerifier
	RangeCorrectness *rp.RangeCorrectnessVerifier
}

// NewVerifier returns a new Verifier instance.
func NewVerifier(inputs, outputs []*math.G1, pp *v1.PublicParams) *Verifier {
	// check if this is an ownership transfer (1 input, 1 output)
	// if so, skip range proof as well-formedness proof is sufficient.
	var rangeCorrectness *rp.RangeCorrectnessVerifier
	if len(inputs) != 1 || len(outputs) != 1 {
		rangeCorrectness = rp.NewRangeCorrectnessVerifier(pp.PedersenGenerators[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])
	}

	return &Verifier{
		PP:               pp,
		TypeAndSum:       NewTypeAndSumVerifier(pp.PedersenGenerators, inputs, outputs, math.Curves[pp.Curve]),
		RangeCorrectness: rangeCorrectness,
	}
}

// Verify checks the validity of a serialized Proof.
func (v *Verifier) Verify(proofRaw []byte) error {
	proof := Proof{}
	err := proof.Deserialize(proofRaw)
	if err != nil {
		return errors.Wrap(err, "invalid transfer proof")
	}
	if err := proof.Validate(v.PP.Curve); err != nil {
		return errors.Wrap(err, "invalid transfer proof")
	}

	// verify well-formedness of inputs and outputs (type and sum)
	tspErr := v.TypeAndSum.Verify(proof.TypeAndSum)
	if tspErr != nil {
		return errors.Wrap(tspErr, "invalid transfer proof")
	}

	// verify range proof if necessary
	if v.RangeCorrectness != nil {
		if proof.RangeCorrectness == nil {
			return ErrMissingRangeProof
		} else {
			commitmentToType := proof.TypeAndSum.CommitmentToType.Copy()
			coms := make([]*math.G1, len(v.TypeAndSum.Outputs))
			for i := range len(v.TypeAndSum.Outputs) {
				coms[i] = v.TypeAndSum.Outputs[i].Copy()
				coms[i].Sub(commitmentToType)
			}
			v.RangeCorrectness.Commitments = coms

			return v.RangeCorrectness.Verify(proof.RangeCorrectness)
		}
	}

	return nil
}

// Prover produces a zero-knowledge proof that a Transfer Action is valid.
type Prover struct {
	TypeAndSum       *TypeAndSumProver
	RangeCorrectness *rp.RangeCorrectnessProver
}

// NewProver returns a new Prover instance.
func NewProver(inputWitness, outputWitness []*token.Metadata, inputs, outputs []*math.G1, pp *v1.PublicParams) (*Prover, error) {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	inW := make([]*token.Metadata, len(inputWitness))
	outW := make([]*token.Metadata, len(outputWitness))
	for i := range inputWitness {
		if inputWitness[i] == nil || inputWitness[i].BlindingFactor == nil {
			return nil, ErrInvalidTokenWitness
		}
		inW[i] = inputWitness[i].Clone()
	}
	values := make([]uint64, len(outputWitness))
	blindingFactors := make([]*math.Zr, len(outputWitness))
	// commit to the type of inputs and outputs
	commitmentToType := pp.PedersenGenerators[0].Mul(c.HashToZr([]byte(inputWitness[0].Type)))

	rand, err := c.Rand()
	if err != nil {
		return nil, err
	}
	typeBF := c.NewRandomZr(rand)
	for i := range outputWitness {
		if outputWitness[i] == nil || outputWitness[i].BlindingFactor == nil {
			return nil, ErrInvalidTokenWitness
		}
		outW[i] = outputWitness[i].Clone()
		values[i], err = outW[i].Value.Uint()
		if err != nil {
			return nil, errors.Wrapf(ErrInvalidTokenWitnessValue, "invalid token witness values [%s]", err)
		}
		blindingFactors[i] = c.ModSub(outW[i].BlindingFactor, typeBF, c.GroupOrder)
	}
	commitmentToType.Add(pp.PedersenGenerators[2].Mul(typeBF))

	p.TypeAndSum = NewTypeAndSumProver(NewTypeAndSumWitness(typeBF, inW, outW, c), pp.PedersenGenerators, inputs, outputs, commitmentToType, c)
	// check if this is an ownership transfer
	// if so, skip range proof, well-formedness proof is enough
	if len(inputWitness) != 1 || len(outputWitness) != 1 {
		coms := make([]*math.G1, len(outputs))
		// The range prover takes as input commitments outputs[i]/commitmentToType
		for i := range outputs {
			coms[i] = outputs[i].Copy()
			coms[i].Sub(commitmentToType)
		}
		p.RangeCorrectness = rp.NewRangeCorrectnessProver(
			coms,
			values,
			blindingFactors,
			pp.PedersenGenerators[1:],
			pp.RangeProofParams.LeftGenerators,
			pp.RangeProofParams.RightGenerators,
			pp.RangeProofParams.P,
			pp.RangeProofParams.Q,
			pp.RangeProofParams.BitLength,
			pp.RangeProofParams.NumberOfRounds,
			math.Curves[pp.Curve],
		)
	}

	return p, nil
}

// Prove produces a serialized zero-knowledge Proof.
func (p *Prover) Prove() ([]byte, error) {
	var wg sync.WaitGroup
	wg.Add(1)

	var tsProof *TypeAndSumProof
	var rangeProof *rp.RangeCorrectness
	if p.RangeCorrectness != nil {
		var err error
		rangeProof, err = p.RangeCorrectness.Prove()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate range proof for transfer")
		}
	}

	tsProof, err := p.TypeAndSum.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer proof")
	}

	proof := &Proof{
		TypeAndSum:       tsProof,
		RangeCorrectness: rangeProof,
	}

	return proof.Serialize()
}
