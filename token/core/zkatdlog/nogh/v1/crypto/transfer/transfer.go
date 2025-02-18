/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"sync"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/pkg/errors"
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

// Verifier verifies if a Action is valid
type Verifier struct {
	TypeAndSum       *TypeAndSumVerifier
	RangeCorrectness *rp.RangeCorrectnessVerifier
}

// NewVerifier returns a Action Verifier as a function of the passed parameters
func NewVerifier(inputs, outputs []*math.G1, pp *v1.PublicParams) *Verifier {
	v := &Verifier{}
	v.TypeAndSum = NewTypeAndSumVerifier(pp.PedersenGenerators, inputs, outputs, math.Curves[pp.Curve])

	// check if this is an ownership transfer
	// if so, skip range proof, well-formedness proof is enough
	if len(inputs) != 1 || len(outputs) != 1 {
		v.RangeCorrectness = rp.NewRangeCorrectnessVerifier(pp.PedersenGenerators[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])
	}

	return v
}

// Prover produces a proof that a Action is valid
type Prover struct {
	TypeAndSum       *TypeAndSumProver
	RangeCorrectness *rp.RangeCorrectnessProver
}

// NewProver returns a Action Prover that corresponds to the passed arguments
func NewProver(inputWitness, outputWitness []*token.TokenDataWitness, inputs, outputs []*math.G1, pp *v1.PublicParams) (*Prover, error) {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	inW := make([]*token.TokenDataWitness, len(inputWitness))
	outW := make([]*token.TokenDataWitness, len(outputWitness))
	for i := 0; i < len(inputWitness); i++ {
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
	for i := 0; i < len(outputWitness); i++ {
		if outputWitness[i] == nil || outputWitness[i].BlindingFactor == nil {
			return nil, errors.New("invalid token witness")
		}
		outW[i] = outputWitness[i].Clone()
		values[i] = outW[i].Value
		blindingFactors[i] = c.ModSub(outW[i].BlindingFactor, typeBF, c.GroupOrder)
	}
	commitmentToType.Add(pp.PedersenGenerators[2].Mul(typeBF))

	p.TypeAndSum = NewTypeAndSumProver(NewTypeAndSumWitness(typeBF, inW, outW, c), pp.PedersenGenerators, inputs, outputs, commitmentToType, c)
	// check if this is an ownership transfer
	// if so, skip range proof, well-formedness proof is enough
	if len(inputWitness) != 1 || len(outputWitness) != 1 {
		coms := make([]*math.G1, len(outputs))
		// The range prover takes as input commitments outputs[i]/commitmentToType
		for i := 0; i < len(outputs); i++ {
			coms[i] = outputs[i].Copy()
			coms[i].Sub(commitmentToType)
		}
		p.RangeCorrectness = rp.NewRangeCorrectnessProver(coms, values, blindingFactors, pp.PedersenGenerators[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])

	}
	return p, nil
}

// Serialize marshals Proof
func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize unmarshals Proof
func (p *Proof) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, p)
}

// Prove produces a serialized Proof
func (p *Prover) Prove() ([]byte, error) {
	var wg sync.WaitGroup
	wg.Add(1)

	var tsProof *TypeAndSumProof
	var rangeProof *rp.RangeCorrectness
	var tsErr, rangeErr error

	go func() {
		defer wg.Done()
		if p.RangeCorrectness != nil {
			rangeProof, rangeErr = p.RangeCorrectness.Prove()
		}
	}()

	tsProof, tsErr = p.TypeAndSum.Prove()

	wg.Wait()

	if tsErr != nil {
		return nil, errors.Wrapf(tsErr, "failed to generate transfer proof")
	}

	if rangeErr != nil {
		return nil, errors.Wrapf(rangeErr, "failed to generate range proof for transfer")
	}

	proof := &Proof{
		TypeAndSum:       tsProof,
		RangeCorrectness: rangeProof,
	}

	return proof.Serialize()
}

// Verify checks validity of serialized Proof
func (v *Verifier) Verify(proof []byte) error {
	tp := Proof{}
	err := tp.Deserialize(proof)
	if err != nil {
		return errors.Wrap(err, "invalid transfer proof")
	}
	if tp.TypeAndSum == nil {
		return errors.New("invalid transfer proof")
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var tspErr, rangeErr error

	// verify well-formedness of inputs and outputs
	tspErr = v.TypeAndSum.Verify(tp.TypeAndSum)

	go func() {
		defer wg.Done()
		// verify range proof
		if v.RangeCorrectness != nil {
			if tp.RangeCorrectness == nil {
				rangeErr = errors.New("invalid transfer proof")
			} else {
				commitmentToType := tp.TypeAndSum.CommitmentToType.Copy()
				coms := make([]*math.G1, len(v.TypeAndSum.Outputs))
				for i := 0; i < len(v.TypeAndSum.Outputs); i++ {
					coms[i] = v.TypeAndSum.Outputs[i].Copy()
					coms[i].Sub(commitmentToType)
				}
				v.RangeCorrectness.Commitments = coms
				rangeErr = v.RangeCorrectness.Verify(tp.RangeCorrectness)
			}
		}
	}()

	wg.Wait()

	if tspErr != nil {
		return errors.Wrap(tspErr, "invalid transfer proof")
	}

	return rangeErr
}
