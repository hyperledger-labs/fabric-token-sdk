/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	crypto "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// todo inspection function

// zero knowledge proof for the consistency between inputs and outputs
type WellFormedness struct {
	InputBlindingFactors  []*math.Zr // blinding factor for inputs
	OutputBlindingFactors []*math.Zr // blinding factor for outputs
	InputValues           []*math.Zr
	OutputValues          []*math.Zr
	Type                  *math.Zr
	Sum                   *math.Zr
	Challenge             *math.Zr
}

func (wf *WellFormedness) Serialize() ([]byte, error) {
	return json.Marshal(wf)
}

func (wf *WellFormedness) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, wf)
}

// inputs and outputs witness for zkat proof
type WellFormednessWitness struct {
	inValues           []*math.Zr
	outValues          []*math.Zr
	Type               string
	inBlindingFactors  []*math.Zr
	outBlindingFactors []*math.Zr
}

func NewWellFormednessWitness(in, out []*token.TokenDataWitness) *WellFormednessWitness {
	inValues := make([]*math.Zr, len(in))
	outValues := make([]*math.Zr, len(out))
	inBF := make([]*math.Zr, len(in))
	outBF := make([]*math.Zr, len(out))
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
	witness *WellFormednessWitness
}

func NewWellFormednessProver(witness *WellFormednessWitness, pp []*math.G1, inputs []*math.G1, outputs []*math.G1, c *math.Curve) *WellFormednessProver {
	verifier := NewWellFormednessVerifier(pp, inputs, outputs, c)
	return &WellFormednessProver{witness: witness, WellFormednessVerifier: verifier}
}

func NewWellFormednessVerifier(pp []*math.G1, inputs []*math.G1, outputs []*math.G1, c *math.Curve) *WellFormednessVerifier {
	return &WellFormednessVerifier{Inputs: inputs, Outputs: outputs, SchnorrVerifier: &crypto.SchnorrVerifier{PedParams: pp, Curve: c}}
}

// SchnorrVerifier for input output correctness
type WellFormednessVerifier struct {
	*crypto.SchnorrVerifier
	Inputs  []*math.G1
	Outputs []*math.G1
}

// Randomness used in proof generation
type WellFormednessRandomness struct {
	inValues  []*math.Zr
	inBF      []*math.Zr
	outValues []*math.Zr
	outBF     []*math.Zr
	Type      *math.Zr
	sum       *math.Zr
}

// Commitments to the randomness in the proof
type WellFormednessCommitments struct {
	Inputs    []*math.G1
	Outputs   []*math.G1
	InputSum  *math.G1
	OutputSum *math.G1
}

// Prove returns zero-knowledge proof for a token transfer
func (p *WellFormednessProver) Prove() ([]byte, error) {
	if len(p.witness.inValues) != len(p.Inputs) || len(p.witness.inBlindingFactors) != len(p.Inputs) || len(p.witness.outValues) != len(p.Outputs) || len(p.witness.outBlindingFactors) != len(p.Outputs) {
		return nil, errors.Errorf("cannot compute transfer proof: malformed witness")
	}
	commitments, randomness, err := p.computeCommitments()
	if err != nil {
		return nil, err
	}

	chal := p.SchnorrVerifier.ComputeChallenge(crypto.GetG1Array(commitments.Inputs, []*math.G1{commitments.InputSum}, commitments.Outputs, []*math.G1{commitments.OutputSum},
		p.Inputs, p.Outputs))
	iop, err := p.computeProof(randomness, chal)
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

func (v *WellFormednessVerifier) parseProof(tokens []*math.G1, values []*math.Zr, randomness []*math.Zr, ttype *math.Zr, sum *math.Zr) ([]*crypto.SchnorrProof, error) {
	if len(values) != len(tokens) || len(randomness) != len(tokens) {
		return nil, errors.Errorf("failed to parse proof ")
	}
	zkps := make([]*crypto.SchnorrProof, len(tokens)+1)
	aggregate := v.Curve.NewG1()
	for i := 0; i < len(tokens); i++ {
		zkps[i] = &crypto.SchnorrProof{}
		zkps[i].Proof = make([]*math.Zr, 3)
		zkps[i].Proof[0] = ttype
		zkps[i].Proof[1] = values[i]
		zkps[i].Proof[2] = randomness[i]
		zkps[i].Statement = tokens[i]
		aggregate.Add(tokens[i])
	}
	zkps[len(tokens)] = &crypto.SchnorrProof{}
	zkps[len(tokens)].Proof = make([]*math.Zr, 3)
	zkps[len(tokens)].Proof[0] = v.Curve.ModMul(ttype, v.Curve.NewZrFromInt(int64(len(tokens))), v.Curve.GroupOrder)
	zkps[len(tokens)].Proof[1] = sum
	zkps[len(tokens)].Proof[2] = crypto.Sum(randomness, v.Curve)
	zkps[len(tokens)].Statement = aggregate

	return zkps, nil
}

func (p *WellFormednessProver) computeProof(randomness *WellFormednessRandomness, chal *math.Zr) (*WellFormedness, error) {
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
	sp = &crypto.SchnorrProver{Witness: []*math.Zr{p.Curve.HashToZr([]byte(p.witness.Type))}, Randomness: []*math.Zr{randomness.Type}, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	typeProof, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the type of transferred tokens")
	}
	wf.Type = typeProof[0]

	// generate zkat proof for the sum of input/output Values
	sum := crypto.Sum(p.witness.inValues, p.Curve)

	sp = &crypto.SchnorrProver{Witness: []*math.Zr{sum}, Randomness: []*math.Zr{randomness.sum}, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	sumProof, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the sum of transferred tokens")
	}

	wf.Sum = sumProof[0]
	wf.Challenge = chal
	return wf, nil
}

func (p *WellFormednessProver) computeCommitments() (*WellFormednessCommitments, *WellFormednessRandomness, error) {
	if len(p.PedParams) != 3 {
		return nil, nil, errors.Errorf("proof generation failed: invalid public parameters")
	}

	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, errors.Errorf("proof generation failed: failed to get random generator")
	}

	randomness := &WellFormednessRandomness{}
	randomness.Type = p.Curve.NewRandomZr(rand) // blindingFactors for type
	Q := p.PedParams[0].Mul(randomness.Type)    // commitment to for type

	randomness.inValues = make([]*math.Zr, len(p.Inputs))
	randomness.inBF = make([]*math.Zr, len(p.Inputs))

	commitments := &WellFormednessCommitments{}
	commitments.Inputs = make([]*math.G1, len(p.Inputs))
	// commitment to sum of inputs, sum of types and sum of blindingFactors
	commitments.InputSum = p.Curve.NewG1()
	for i := 0; i < len(p.Inputs); i++ {
		// randomness for value
		randomness.inValues[i] = p.Curve.NewRandomZr(rand)
		// randomness for blinding factor
		randomness.inBF[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		commitments.Inputs[i] = p.PedParams[1].Mul(randomness.inValues[i])
		commitments.Inputs[i].Add(Q)
		P := p.PedParams[2].Mul(randomness.inBF[i])
		commitments.Inputs[i].Add(P)
		commitments.InputSum.Add(P)
	}
	randomness.sum = p.Curve.NewRandomZr(rand) // blindingFactors for sum
	commitments.InputSum.Add(p.PedParams[1].Mul(randomness.sum))
	commitments.InputSum.Add(Q.Mul(p.Curve.NewZrFromInt(int64(len(p.Inputs)))))

	// preparing commitments for outputs
	randomness.outValues = make([]*math.Zr, len(p.Outputs))
	randomness.outBF = make([]*math.Zr, len(p.Outputs))

	commitments.Outputs = make([]*math.G1, len(p.Outputs))
	commitments.OutputSum = p.Curve.NewG1()
	commitments.OutputSum.Add(p.PedParams[1].Mul(randomness.sum))
	commitments.OutputSum.Add(Q.Mul(p.Curve.NewZrFromInt(int64(len(p.Outputs)))))
	for i := 0; i < len(p.Outputs); i++ {
		// generate randomness
		randomness.outValues[i] = p.Curve.NewRandomZr(rand)
		randomness.outBF[i] = p.Curve.NewRandomZr(rand)
		// compute commitment
		commitments.Outputs[i] = p.PedParams[1].Mul(randomness.outValues[i])
		commitments.Outputs[i].Add(Q)
		P := p.PedParams[2].Mul(randomness.outBF[i])
		commitments.Outputs[i].Add(P)
		commitments.OutputSum.Add(P)
	}
	return commitments, randomness, nil
}
