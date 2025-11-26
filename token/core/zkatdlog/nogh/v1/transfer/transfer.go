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

// Proof is a zero-knowledge proof that shows that a Action is valid
type Proof struct {
	// proof that inputs and outputs in a Transfer Action are well-formed
	// inputs and outputs have the same total value
	// inputs and outputs have the same type
	TypeAndSum *TypeAndSumProof
	// Proof that the outputs have value in the authorized range
	RangeCorrectness *rp.RangeCorrectness
}

// Serialize marshals Proof
func (p *Proof) Serialize() ([]byte, error) {
	return asn1.Marshal[asn1.Serializer](p.TypeAndSum, p.RangeCorrectness)
}

// Deserialize unmarshals Proof
func (p *Proof) Deserialize(bytes []byte) error {
	p.TypeAndSum = &TypeAndSumProof{}
	p.RangeCorrectness = &rp.RangeCorrectness{}
	return asn1.Unmarshal[asn1.Serializer](bytes, p.TypeAndSum, p.RangeCorrectness)
}

func (p *Proof) Validate(curve math.CurveID) error {
	if p.TypeAndSum == nil {
		return errors.New("invalid transfer proof")
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

// Verifier verifies if a Action is valid
type Verifier struct {
	PP               *v1.PublicParams
	TypeAndSum       *TypeAndSumVerifier
	RangeCorrectness *rp.RangeCorrectnessVerifier
}

// NewVerifier returns a Action Verifier as a function of the passed parameters
func NewVerifier(inputs, outputs []*math.G1, pp *v1.PublicParams) *Verifier {
	// check if this is an ownership transfer
	// if so, skip range proof, well-formedness proof is enough
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

// Verify checks validity of serialized Proof
func (v *Verifier) Verify(proofRaw []byte) error {
	proof := Proof{}
	err := proof.Deserialize(proofRaw)
	if err != nil {
		return errors.Wrap(err, "invalid transfer proof")
	}
	if err := proof.Validate(v.PP.Curve); err != nil {
		return errors.Wrap(err, "invalid transfer proof")
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// verify well-formedness of inputs and outputs
	tspErr := v.TypeAndSum.Verify(proof.TypeAndSum)
	if tspErr != nil {
		return errors.Wrap(tspErr, "invalid transfer proof")
	}

	// verify range proof
	if v.RangeCorrectness != nil {
		if proof.RangeCorrectness == nil {
			return errors.New("invalid transfer proof")
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

// Prover produces a proof that a Action is valid
type Prover struct {
	TypeAndSum       *TypeAndSumProver
	RangeCorrectness *rp.RangeCorrectnessProver
}

// NewProver returns a Action Prover that corresponds to the passed arguments
func NewProver(inputWitness, outputWitness []*token.Metadata, inputs, outputs []*math.G1, pp *v1.PublicParams) (*Prover, error) {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	inW := make([]*token.Metadata, len(inputWitness))
	outW := make([]*token.Metadata, len(outputWitness))
	for i := range inputWitness {
		if inputWitness[i] == nil || inputWitness[i].BlindingFactor == nil {
			return nil, errors.New("invalid token witness")
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
			return nil, errors.New("invalid token witness")
		}
		outW[i] = outputWitness[i].Clone()
		values[i], err = outW[i].Value.Uint()
		if err != nil {
			return nil, errors.Wrapf(err, "invalid token witness values")
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

// Prove produces a serialized Proof
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
