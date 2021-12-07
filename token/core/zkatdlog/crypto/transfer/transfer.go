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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	rangeproof "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/range"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// zkat proof of transfer correctness
type Proof struct {
	WellFormedness   []byte // input output correctness proof
	RangeCorrectness []byte // range correctness proof
}

// verifier for zkat transfer
type Verifier struct {
	WellFormedness   common.Verifier
	RangeCorrectness common.Verifier
}

// prover for zkat transfer
type Prover struct {
	WellFormedness   common.Prover
	RangeCorrectness common.Prover
}

func NewProver(inputwitness, outputwitness []*token.TokenDataWitness, inputs, outputs []*math.G1, pp *crypto.PublicParams) *Prover {
	p := &Prover{}

	inW := make([]*token.TokenDataWitness, len(inputwitness))
	outW := make([]*token.TokenDataWitness, len(outputwitness))

	for i := 0; i < len(inputwitness); i++ {
		inW[i] = inputwitness[i].Clone()
	}

	for i := 0; i < len(outputwitness); i++ {
		outW[i] = outputwitness[i].Clone()
	}
	if len(inputwitness) != 1 || len(outputwitness) != 1 {
		p.RangeCorrectness = rangeproof.NewProver(outW, outputs, pp.RangeProofParams.SignedValues, pp.RangeProofParams.Exponent, pp.ZKATPedParams, pp.RangeProofParams.SignPK, pp.P, pp.RangeProofParams.Q, math.Curves[pp.Curve])
	}
	wfw := NewWellFormednessWitness(inW, outW)
	p.WellFormedness = NewWellFormednessProver(wfw, pp.ZKATPedParams, inputs, outputs, math.Curves[pp.Curve])
	return p
}

func NewVerifier(inputs, outputs []*math.G1, pp *crypto.PublicParams) *Verifier {
	v := &Verifier{}
	if len(inputs) != 1 || len(outputs) != 1 {
		v.RangeCorrectness = rangeproof.NewVerifier(outputs, uint64(len(pp.RangeProofParams.SignedValues)), pp.RangeProofParams.Exponent, pp.ZKATPedParams, pp.RangeProofParams.SignPK, pp.P, pp.RangeProofParams.Q, math.Curves[pp.Curve])
	}
	v.WellFormedness = NewWellFormednessVerifier(pp.ZKATPedParams, inputs, outputs, math.Curves[pp.Curve])

	return v
}

func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Proof) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, p)
}

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

func (v *Verifier) Verify(proof []byte) error {
	tp := *&Proof{}
	err := tp.Deserialize(proof)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof: cannot parse proof")
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
		return wfErr
	}

	return rangeErr
}

func (w *WellFormednessWitness) GetInValues() []*math.Zr {
	return w.inValues
}

func (w *WellFormednessWitness) GetOutValues() []*math.Zr {
	return w.outValues
}

func (w *WellFormednessWitness) GetOutBlindingFators() []*math.Zr {
	return w.outBlindingFactors
}

func (w *WellFormednessWitness) GetInBlindingFators() []*math.Zr {
	return w.inBlindingFactors
}
