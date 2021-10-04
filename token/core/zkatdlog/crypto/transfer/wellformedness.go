/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer

import (
	"encoding/json"

	bn256 "github.com/IBM/mathlib"
	crypto "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// todo inspection function

// zero knowledge proof for the consistency between inputs and outputs
type WellFormedness struct {
	InputBlindingFactors  []*bn256.Zr // blinding factor for inputs
	OutputBlindingFactors []*bn256.Zr // blinding factor for outputs
	InputValues           []*bn256.Zr
	OutputValues          []*bn256.Zr
	Type                  *bn256.Zr
	Sum                   *bn256.Zr
	Challenge             *bn256.Zr
}

func (wf *WellFormedness) Serialize() ([]byte, error) {
	return json.Marshal(wf)
}

func (wf *WellFormedness) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, wf)
}

// inputs and outputs witness for zkat proof
type WellFormednessWitness struct {
	inValues           []*bn256.Zr
	outValues          []*bn256.Zr
	Type               string
	inBlindingFactors  []*bn256.Zr
	outBlindingFactors []*bn256.Zr
}

func NewWellFormednessWitness(in, out []*token.TokenDataWitness) *WellFormednessWitness {
	inValues := make([]*bn256.Zr, len(in))
	outValues := make([]*bn256.Zr, len(out))
	inBF := make([]*bn256.Zr, len(in))
	outBF := make([]*bn256.Zr, len(out))
	for i := 0; i < len(in); i++ {
		inValues[i] = in[i].Value
		inBF[i] = in[i].BlindingFactor
	}
	for i := 0; i < len(out); i++ {
		outValues[i] = out[i].Value
		outBF[i] = out[i].BlindingFactor
	}
	return &WellFormednessWitness{inValues: inValues, outValues: outValues, Type: in[0].Type, inBlindingFactors: inBF, outBlindingFactors: outBF}
}

// Prover for input output correctness
type WellFormednessProver struct {
	*WellFormednessVerifier
	witness     *WellFormednessWitness
	randomness  *WellFormednessRandomness
	Commitments *WellFormednessCommitments
}

func NewWellFormednessProver(witness *WellFormednessWitness, pp []*bn256.G1, inputs []*bn256.G1, outputs []*bn256.G1, c *bn256.Curve) *WellFormednessProver {
	verifier := NewWellFormednessVerifier(pp, inputs, outputs, c)
	return &WellFormednessProver{witness: witness, WellFormednessVerifier: verifier}
}

func NewWellFormednessVerifier(pp []*bn256.G1, inputs []*bn256.G1, outputs []*bn256.G1, c *bn256.Curve) *WellFormednessVerifier {
	return &WellFormednessVerifier{Inputs: inputs, Outputs: outputs, SchnorrVerifier: &crypto.SchnorrVerifier{PedParams: pp, Curve: c}}
}

// SchnorrVerifier for input output correctness
type WellFormednessVerifier struct {
	*crypto.SchnorrVerifier
	Inputs  []*bn256.G1
	Outputs []*bn256.G1
}

// Randomness used in proof generation
type WellFormednessRandomness struct {
	inValues  []*bn256.Zr
	inBF      []*bn256.Zr
	outValues []*bn256.Zr
	outBF     []*bn256.Zr
	Type      *bn256.Zr
	sum       *bn256.Zr
}

// Commitments to the randomness in the proof
type WellFormednessCommitments struct {
	Inputs    []*bn256.G1
	Outputs   []*bn256.G1
	InputSum  *bn256.G1
	OutputSum *bn256.G1
}

// Prove returns zero-knowledge proof for a token transfer
func (p *WellFormednessProver) Prove() ([]byte, error) {
	if len(p.witness.inValues) != len(p.Inputs) || len(p.witness.inBlindingFactors) != len(p.Inputs) || len(p.witness.outValues) != len(p.Outputs) || len(p.witness.outBlindingFactors) != len(p.Outputs) {
		return nil, errors.Errorf("cannot compute transfer proof: malformed witness")
	}
	err := p.computeCommitments()
	if err != nil {
		return nil, err
	}

	chal := p.SchnorrVerifier.ComputeChallenge(crypto.GetG1Array(p.Commitments.Inputs, []*bn256.G1{p.Commitments.InputSum}, p.Commitments.Outputs, []*bn256.G1{p.Commitments.OutputSum},
		p.Inputs, p.Outputs))
	iop, err := p.computeProof(p.randomness, chal)
	if err != nil {
		return nil, err
	}
	return iop.Serialize()
}

// Verify returns an error when zktp is not a valid transfer proof
func (v *WellFormednessVerifier) Verify(p []byte) error {
	iop := &WellFormedness{}
	err := iop.Deserialize(p)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof: cannot parse proof")
	}
	zkps, err := v.parseProof(v.Inputs, iop.InputValues, iop.InputBlindingFactors, iop.Type, iop.Sum)
	inCommitments := v.RecomputeCommitments(zkps, iop.Challenge)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}
	zkps, err = v.parseProof(v.Outputs, iop.OutputValues, iop.OutputBlindingFactors, iop.Type, iop.Sum)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}
	outCommitments := v.RecomputeCommitments(zkps, iop.Challenge)

	chal := v.SchnorrVerifier.ComputeChallenge(crypto.GetG1Array(inCommitments, outCommitments, v.Inputs, v.Outputs))
	if !chal.Equals(iop.Challenge) {
		return errors.Errorf("invalid zero-knowledge transfer")
	}
	return nil
}

func (v *WellFormednessVerifier) parseProof(tokens []*bn256.G1, values []*bn256.Zr, randomness []*bn256.Zr, ttype *bn256.Zr, sum *bn256.Zr) ([]*crypto.SchnorrProof, error) {
	if len(values) != len(tokens) || len(randomness) != len(tokens) {
		return nil, errors.Errorf("failed to parse proof ")
	}
	zkps := make([]*crypto.SchnorrProof, len(tokens)+1)
	aggregate := v.Curve.NewG1()
	for i := 0; i < len(tokens); i++ {
		zkps[i] = &crypto.SchnorrProof{}
		zkps[i].Proof = make([]*bn256.Zr, 3)
		zkps[i].Proof[0] = ttype
		zkps[i].Proof[1] = values[i]
		zkps[i].Proof[2] = randomness[i]
		zkps[i].Statement = tokens[i]
		aggregate.Add(tokens[i])
	}
	zkps[len(tokens)] = &crypto.SchnorrProof{}
	zkps[len(tokens)].Proof = make([]*bn256.Zr, 3)
	zkps[len(tokens)].Proof[0] = v.Curve.ModMul(ttype, v.Curve.NewZrFromInt(int64(len(tokens))), v.Curve.GroupOrder)
	zkps[len(tokens)].Proof[1] = sum
	zkps[len(tokens)].Proof[2] = crypto.Sum(randomness, v.Curve)
	zkps[len(tokens)].Statement = aggregate

	return zkps, nil
}

func (p *WellFormednessProver) computeProof(randomness *WellFormednessRandomness, chal *bn256.Zr) (*WellFormedness, error) {
	if len(p.witness.inValues) != len(p.witness.inBlindingFactors) || len(p.witness.outValues) != len(p.witness.outBlindingFactors) {
		return nil, errors.Errorf("proof generation for transfer failed: invalid witness")
	}
	if len(randomness.inValues) != len(p.witness.inValues) || len(randomness.outValues) != len(p.witness.outValues) || len(randomness.outBF) != len(p.witness.outBlindingFactors) || len(randomness.inBF) != len(p.witness.inBlindingFactors) {
		return nil, errors.Errorf("proof generation for transfer failed: invalid blindingFactors")
	}

	wf := &WellFormedness{}
	var err error
	// generate zkat proof for input Values
	sp := &crypto.SchnorrProver{Witness: p.witness.inValues, Randomness: randomness.inValues, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.InputValues, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for input Values")
	}

	// generate zkat proof for inputs' blindingFactors
	sp = &crypto.SchnorrProver{Witness: p.witness.inBlindingFactors, Randomness: randomness.inBF, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.InputBlindingFactors, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the blindingFactors in the inputs")
	}

	// generate zkat proof for output Values
	sp = &crypto.SchnorrProver{Witness: p.witness.outValues, Randomness: randomness.outValues, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.OutputValues, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for output Values")
	}

	// generate zkat proof for blindingFactors in outputs
	sp = &crypto.SchnorrProver{Witness: p.witness.outBlindingFactors, Randomness: randomness.outBF, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.OutputBlindingFactors, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the blindingFactors in the outputs")
	}

	// generate zkat proof for token type
	sp = &crypto.SchnorrProver{Witness: []*bn256.Zr{p.Curve.HashToZr([]byte(p.witness.Type))}, Randomness: []*bn256.Zr{randomness.Type}, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	typeProof, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the type of transferred tokens")

	}
	wf.Type = typeProof[0]

	// generate zkat proof for the sum of input/output Values
	sum := crypto.Sum(p.witness.inValues, p.Curve)

	sp = &crypto.SchnorrProver{Witness: []*bn256.Zr{sum}, Randomness: []*bn256.Zr{randomness.sum}, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	sumProof, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the sum of transferred tokens")
	}

	wf.Sum = sumProof[0]
	wf.Challenge = chal
	return wf, nil
}

func (p *WellFormednessProver) computeCommitments() error {
	if len(p.PedParams) != 3 {
		return errors.Errorf("proof generation failed: invalid public parameters")
	}

	rand, err := p.Curve.Rand()
	if err != nil {
		return errors.Errorf("proof generation failed: failed to get random generator")
	}

	p.randomness = &WellFormednessRandomness{}
	p.randomness.Type = p.Curve.NewRandomZr(rand) // blindingFactors for type
	Q := p.PedParams[0].Mul(p.randomness.Type)    // commitment to for type

	p.randomness.inValues = make([]*bn256.Zr, len(p.Inputs))
	p.randomness.inBF = make([]*bn256.Zr, len(p.Inputs))

	p.Commitments = &WellFormednessCommitments{}
	p.Commitments.Inputs = make([]*bn256.G1, len(p.Inputs))
	// commitment to sum of inputs, sum of types and sum of blindingFactors
	p.Commitments.InputSum = p.Curve.NewG1()
	for i := 0; i < len(p.Inputs); i++ {
		// randomness for value
		p.randomness.inValues[i] = p.Curve.NewRandomZr(rand)
		// randomness for blinding factor
		p.randomness.inBF[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		p.Commitments.Inputs[i] = p.PedParams[1].Mul(p.randomness.inValues[i])
		p.Commitments.Inputs[i].Add(Q)
		P := p.PedParams[2].Mul(p.randomness.inBF[i])
		p.Commitments.Inputs[i].Add(P)
		p.Commitments.InputSum.Add(P)
	}
	p.randomness.sum = p.Curve.NewRandomZr(rand) // blindingFactors for sum
	p.Commitments.InputSum.Add(p.PedParams[1].Mul(p.randomness.sum))
	p.Commitments.InputSum.Add(Q.Mul(p.Curve.NewZrFromInt(int64(len(p.Inputs)))))

	// preparing commitments for outputs
	p.randomness.outValues = make([]*bn256.Zr, len(p.Outputs))
	p.randomness.outBF = make([]*bn256.Zr, len(p.Outputs))

	p.Commitments.Outputs = make([]*bn256.G1, len(p.Outputs))
	p.Commitments.OutputSum = p.Curve.NewG1()
	p.Commitments.OutputSum.Add(p.PedParams[1].Mul(p.randomness.sum))
	p.Commitments.OutputSum.Add(Q.Mul(p.Curve.NewZrFromInt(int64(len(p.Outputs)))))
	for i := 0; i < len(p.Outputs); i++ {
		// generate randomness
		p.randomness.outValues[i] = p.Curve.NewRandomZr(rand)
		p.randomness.outBF[i] = p.Curve.NewRandomZr(rand)
		// compute commitment
		p.Commitments.Outputs[i] = p.PedParams[1].Mul(p.randomness.outValues[i])
		p.Commitments.Outputs[i].Add(Q)
		P := p.PedParams[2].Mul(p.randomness.outBF[i])
		p.Commitments.Outputs[i].Add(P)
		p.Commitments.OutputSum.Add(P)
	}
	return nil
}
