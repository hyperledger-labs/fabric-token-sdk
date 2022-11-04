/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"encoding/json"
	"sync"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	rangeproof "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/range"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// Proof is a zero-knowledge proof that shows that a TransferAction is valid
type Proof struct {
	// proof that inputs and outputs in a Transfer Action are well-formed
	// inputs and outputs have the same total value
	// inputs and outputs have the same type
	WellFormedness []byte
	// Proof that the outputs have value in the authorized range
	RangeCorrectness []byte
}

// Verifier verifies if a TransferAction is valid
type Verifier struct {
	WellFormedness   *WellFormednessVerifier
	RangeCorrectness *rangeproof.Verifier
}

// Prover produces a proof that a TransferAction is valid
type Prover struct {
	WellFormedness   *WellFormednessProver
	RangeCorrectness *rangeproof.Prover
}

// NewProver returns a TransferAction Prover that corresponds to the passed arguments
func NewProver(inputWitness, outputWitness []*token.TokenDataWitness, inputs, outputs []*math.G1, pp *crypto.PublicParams) *Prover {
	p := &Prover{}

	inW := make([]*token.TokenDataWitness, len(inputWitness))
	outW := make([]*token.TokenDataWitness, len(outputWitness))

	for i := 0; i < len(inputWitness); i++ {
		inW[i] = inputWitness[i].Clone()
	}

	for i := 0; i < len(outputWitness); i++ {
		outW[i] = outputWitness[i].Clone()
	}
	// check if this is an ownership transfer
	// if so, skip range proof, well-formedness proof is enough
	if len(inputWitness) != 1 || len(outputWitness) != 1 {
		p.RangeCorrectness = rangeproof.NewProver(
			outW,
			outputs,
			pp.RangeProofParams.SignedValues,
			int(pp.RangeProofParams.Exponent),
			pp.PedParams,
			pp.RangeProofParams.SignPK,
			pp.PedGen,
			pp.RangeProofParams.Q,
			math.Curves[pp.Curve],
		)
	}
	wfw := NewWellFormednessWitness(inW, outW)
	p.WellFormedness = NewWellFormednessProver(wfw, pp.PedParams, inputs, outputs, math.Curves[pp.Curve])
	return p
}

// NewVerifier returns a TransferAction Verifier as a function of the passed parameters
func NewVerifier(inputs, outputs []*math.G1, pp *crypto.PublicParams) *Verifier {
	v := &Verifier{}
	// check if this is an ownership transfer
	// if so, skip range proof, well-formedness proof is enough
	if len(inputs) != 1 || len(outputs) != 1 {
		v.RangeCorrectness = rangeproof.NewVerifier(
			outputs,
			uint64(len(pp.RangeProofParams.SignedValues)),
			int(pp.RangeProofParams.Exponent),
			pp.PedParams,
			pp.RangeProofParams.SignPK,
			pp.PedGen,
			pp.RangeProofParams.Q,
			math.Curves[pp.Curve],
		)
	}
	v.WellFormedness = NewWellFormednessVerifier(pp.PedParams, inputs, outputs, math.Curves[pp.Curve])

	return v
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

	var wfProof, rangeProof []byte
	var wfErr, rangeErr error

	go func() {
		defer wg.Done()
		if p.RangeCorrectness != nil {
			rangeProof, rangeErr = p.RangeCorrectness.Prove()
		}
	}()

	wfProof, wfErr = p.WellFormedness.Prove()

	wg.Wait()

	if wfErr != nil {
		return nil, errors.Wrapf(wfErr, "failed to generate transfer proof")
	}

	if rangeErr != nil {
		return nil, errors.Wrapf(rangeErr, "failed to generate range proof for transfer")
	}

	proof := &Proof{
		WellFormedness:   wfProof,
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

	var wg sync.WaitGroup
	wg.Add(1)

	var wfErr, rangeErr error

	// verify well-formedness of inputs and outputs
	wfErr = v.WellFormedness.Verify(tp.WellFormedness)

	go func() {
		defer wg.Done()
		// verify range proof
		if v.RangeCorrectness != nil {
			rangeErr = v.RangeCorrectness.Verify(tp.RangeCorrectness)
		}
	}()

	wg.Wait()

	if wfErr != nil {
		return errors.Wrap(wfErr, "invalid transfer proof")
	}

	return rangeErr
}
