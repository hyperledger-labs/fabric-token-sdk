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

// WellFormedness is zero-knowledge proof that shows that an array of input Pedersen commitments
//and an array of output Pedersen  commitments have the same total value and the same type

type WellFormedness struct {
	// proof of knowledge of the randomness used in Pedersen commitments in the inputs
	InputBlindingFactors []*math.Zr
	// proof of knowledge of the randomness used in Pedersen commitments in the outputs
	OutputBlindingFactors []*math.Zr
	// proof of knowledge of the values encoded in the Pedersen commitments in the inputs
	InputValues []*math.Zr
	// proof of knowledge of the values encoded in the Pedersen commitments in the outputs
	OutputValues []*math.Zr
	// proof of knowledge of the token type encoded in both inputs and outputs
	Type *math.Zr
	// proof of knowledge of the sum of inputs and the sum of outputs
	// sum of inputs equal sum of outputs
	Sum *math.Zr
	// challenge used in proof
	Challenge *math.Zr
}

// Serialize marshals WellFormedness
func (wf *WellFormedness) Serialize() ([]byte, error) {
	return json.Marshal(wf)
}

// Deserialize un-marshals WellFormedness
func (wf *WellFormedness) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, wf)
}

// WellFormednessWitness contains the secret information used to produce WellFormedness
type WellFormednessWitness struct {
	// inValues carries the values of the inputs
	inValues []*math.Zr
	// outValues carries the values of the outputs
	outValues []*math.Zr
	// Type is the token type of inputs and outputs
	Type string
	// inBlindingFactors carries the randomness used to compute the Pedersen commitments
	// in inputs
	inBlindingFactors []*math.Zr
	// outBlindingFactors carries the randomness used to compute the Pedersen commitments
	// in outputs
	outBlindingFactors []*math.Zr
}

// NewWellFormednessWitness returns a WellFormednessWitness as a function of the passed arguments
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

// WellFormednessProver produces a WellFormedness proof
type WellFormednessProver struct {
	*WellFormednessVerifier
	witness *WellFormednessWitness
}

// NewWellFormednessProver returns a NewWellFormednessProver as a function of the passed arguments
func NewWellFormednessProver(witness *WellFormednessWitness, pp []*math.G1, inputs []*math.G1, outputs []*math.G1, c *math.Curve) *WellFormednessProver {
	verifier := NewWellFormednessVerifier(pp, inputs, outputs, c)
	return &WellFormednessProver{witness: witness, WellFormednessVerifier: verifier}
}

// NewWellFormednessVerifier returns a NewWellFormednessVerifier as a function of the passed arguments
func NewWellFormednessVerifier(pp []*math.G1, inputs []*math.G1, outputs []*math.G1, c *math.Curve) *WellFormednessVerifier {
	return &WellFormednessVerifier{Inputs: inputs, Outputs: outputs, PedParams: pp, Curve: c}
}

// WellFormednessVerifier checks the validity of WellFormedness
type WellFormednessVerifier struct {
	// PedParams corresponds to the generators used to compute Pedersen commitments
	// (g_1, g_2, h)
	PedParams []*math.G1
	// Curve is the elliptic curve in which Pedersen commitments are computed
	Curve *math.Curve
	// Inputs are Pedersen commitments to (Type, Value) of the inputs to be spent
	Inputs []*math.G1
	// Outputs are Pedersen commitments to (Type, Value) of the outputs to be created
	// after the transfer
	Outputs []*math.G1
}

// WellFormednessRandomness is the randomness used in the generation of WellFormedness
type WellFormednessRandomness struct {
	inValues  []*math.Zr
	inBF      []*math.Zr
	outValues []*math.Zr
	outBF     []*math.Zr
	Type      *math.Zr
	sum       *math.Zr
}

// WellFormednessCommitments are commitments to the randomness used in WellFormedness
type WellFormednessCommitments struct {
	Inputs    []*math.G1
	Outputs   []*math.G1
	InputSum  *math.G1
	OutputSum *math.G1
}

// Prove returns a serialized WellFormedness proof
func (p *WellFormednessProver) Prove() ([]byte, error) {
	// check if witness length match inputs and outputs length
	if len(p.witness.inValues) != len(p.Inputs) || len(p.witness.inBlindingFactors) != len(p.Inputs) || len(p.witness.outValues) != len(p.Outputs) || len(p.witness.outBlindingFactors) != len(p.Outputs) {
		return nil, errors.Errorf("cannot compute transfer proof: malformed witness")
	}
	// generate randomness for the proof and compute the corresponding commitments
	commitments, randomness, err := p.computeCommitments()
	if err != nil {
		return nil, err
	}
	// serialize public information
	raw, err := crypto.GetG1Array(commitments.Inputs, []*math.G1{commitments.InputSum}, commitments.Outputs, []*math.G1{commitments.OutputSum}, p.Inputs, p.Outputs).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "cannot compute transfer proof")
	}
	// compute challenge
	chal := p.Curve.HashToZr(raw)
	// compute proofs
	wf, err := p.computeProof(randomness, chal)
	if err != nil {
		return nil, err
	}
	return wf.Serialize()
}

// Verify returns an error when WellFormedness is not a valid
func (v *WellFormednessVerifier) Verify(p []byte) error {
	// deserialize WellFormedness
	wf := &WellFormedness{}
	err := wf.Deserialize(p)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof: cannot parse proof")
	}

	// recompute commitments to proof randomness as a function of the parsed proofs and the challenge
	// for inputs
	zkps, err := v.parseProof(v.Inputs, wf.InputValues, wf.InputBlindingFactors, wf.Type, wf.Sum)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}
	sv := &crypto.SchnorrVerifier{Curve: v.Curve, PedParams: v.PedParams}
	inCommitments, err := sv.RecomputeCommitments(zkps, wf.Challenge)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}

	// for outputs
	zkps, err = v.parseProof(v.Outputs, wf.OutputValues, wf.OutputBlindingFactors, wf.Type, wf.Sum)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}
	outCommitments, err := sv.RecomputeCommitments(zkps, wf.Challenge)
	if err != nil {
		return errors.Wrapf(err, "invalid transfer proof")
	}

	// compute challenge
	raw, err := crypto.GetG1Array(inCommitments, outCommitments, v.Inputs, v.Outputs).Bytes()
	if err != nil {
		return errors.Wrapf(err, "cannot verify transfer proof")
	}
	// check if proof is valid
	if !v.Curve.HashToZr(raw).Equals(wf.Challenge) {
		return errors.Errorf("invalid zero-knowledge transfer")
	}
	return nil
}

// parseProof returns an array of Schnorr proofs
func (v *WellFormednessVerifier) parseProof(tokens []*math.G1, values []*math.Zr, randomness []*math.Zr, ttype *math.Zr, sum *math.Zr) ([]*crypto.SchnorrProof, error) {
	if len(values) != len(tokens) || len(randomness) != len(tokens) {
		return nil, errors.New("failed to parse wellformedness proof ")
	}
	if v.Curve == nil {
		return nil, errors.New("failed to parse wellformedness proof: please initialize curve")

	}
	zkps := make([]*crypto.SchnorrProof, len(tokens)+1)
	// aggregate is the sum of tokens
	aggregate := v.Curve.NewG1()
	for i := 0; i < len(tokens); i++ {
		zkps[i] = &crypto.SchnorrProof{}
		zkps[i].Proof = make([]*math.Zr, 3)
		zkps[i].Proof[0] = ttype
		zkps[i].Proof[1] = values[i]
		zkps[i].Proof[2] = randomness[i]
		zkps[i].Statement = tokens[i]
		if tokens[i] == nil {
			return nil, errors.Errorf("invalid wellformedness proof")
		}
		aggregate.Add(tokens[i])
	}
	// proof of knowledge of the opening of aggregate
	zkps[len(tokens)] = &crypto.SchnorrProof{}
	zkps[len(tokens)].Proof = make([]*math.Zr, 3)
	// proof of ttype * len(tokens)
	zkps[len(tokens)].Proof[0] = v.Curve.ModMul(ttype, v.Curve.NewZrFromInt(int64(len(tokens))), v.Curve.GroupOrder)
	// proof of value of aggregate which corresponds to sum
	zkps[len(tokens)].Proof[1] = sum
	var err error
	// proof of randomness in aggregate which corresponds to the sum of all the randomness
	// used in all tokens
	zkps[len(tokens)].Proof[2], err = crypto.Sum(randomness, v.Curve)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid wellformedness proof")
	}
	zkps[len(tokens)].Statement = aggregate

	return zkps, nil
}

// computeProof compute Wellformedness as a function of the passed randomness and challenge
func (p *WellFormednessProver) computeProof(randomness *WellFormednessRandomness, chal *math.Zr) (*WellFormedness, error) {
	if len(p.witness.inValues) != len(p.witness.inBlindingFactors) || len(p.witness.outValues) != len(p.witness.outBlindingFactors) {
		return nil, errors.Errorf("proof generation for transfer failed: invalid witness")
	}
	if len(randomness.inValues) != len(p.witness.inValues) || len(randomness.outValues) != len(p.witness.outValues) || len(randomness.outBF) != len(p.witness.outBlindingFactors) || len(randomness.inBF) != len(p.witness.inBlindingFactors) {
		return nil, errors.Errorf("proof generation for transfer failed: invalid blindingFactors")
	}

	wf := &WellFormedness{}
	var err error
	// generate zk proof for input values
	sp := &crypto.SchnorrProver{Witness: p.witness.inValues, Randomness: randomness.inValues, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.InputValues, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for input Values")
	}

	// generate zk proof for randomness used to compute inputs
	sp = &crypto.SchnorrProver{Witness: p.witness.inBlindingFactors, Randomness: randomness.inBF, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.InputBlindingFactors, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the blindingFactors in the inputs")
	}

	// generate zk proof for output values
	sp = &crypto.SchnorrProver{Witness: p.witness.outValues, Randomness: randomness.outValues, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.OutputValues, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for output Values")
	}

	// generate zk proof for randomness used to compute outputs
	sp = &crypto.SchnorrProver{Witness: p.witness.outBlindingFactors, Randomness: randomness.outBF, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	wf.OutputBlindingFactors, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the blindingFactors in the outputs")
	}

	// generate zk proof for token type
	sp = &crypto.SchnorrProver{Witness: []*math.Zr{p.Curve.HashToZr([]byte(p.witness.Type))}, Randomness: []*math.Zr{randomness.Type}, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	typeProof, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the type of transferred tokens")
	}
	wf.Type = typeProof[0]

	// generate zk proof for the sum of input/output values
	sum, err := crypto.Sum(p.witness.inValues, p.Curve)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the sum of transferred tokens")
	}

	sp = &crypto.SchnorrProver{Witness: []*math.Zr{sum}, Randomness: []*math.Zr{randomness.sum}, Challenge: chal, SchnorrVerifier: &crypto.SchnorrVerifier{Curve: p.Curve}}
	sumProof, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the sum of transferred tokens")
	}

	wf.Sum = sumProof[0]
	wf.Challenge = chal
	return wf, nil
}

// computeCommitments returns the randomness used in WellFormedness proof and the corresponding commitments
func (p *WellFormednessProver) computeCommitments() (*WellFormednessCommitments, *WellFormednessRandomness, error) {
	if len(p.PedParams) != 3 {
		return nil, nil, errors.New("invalid public parameters")
	}

	if p.Curve == nil {
		return nil, nil, errors.New("please initialize curve")
	}
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, errors.New("failed to get random generator")
	}

	// produce randomness for the WellFormedness proof
	randomness := &WellFormednessRandomness{}
	randomness.Type = p.Curve.NewRandomZr(rand) // randomness to prove token type
	if p.PedParams[0] == nil || p.PedParams[1] == nil || p.PedParams[2] == nil {
		return nil, nil, errors.New("please provide non-nil Pedersen parameters")
	}
	Q := p.PedParams[0].Mul(randomness.Type) // commitment to randomness for type

	// for inputs
	randomness.inValues = make([]*math.Zr, len(p.Inputs))
	randomness.inBF = make([]*math.Zr, len(p.Inputs))

	commitments := &WellFormednessCommitments{}
	commitments.Inputs = make([]*math.G1, len(p.Inputs))
	// commitment to the randomness for the sum of inputs
	commitments.InputSum = p.Curve.NewG1()
	for i := 0; i < len(p.Inputs); i++ {
		// randomness to prove input values
		randomness.inValues[i] = p.Curve.NewRandomZr(rand)
		// randomness to prove input blinding factors
		randomness.inBF[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		commitments.Inputs[i] = p.PedParams[1].Mul(randomness.inValues[i])
		commitments.Inputs[i].Add(Q)
		P := p.PedParams[2].Mul(randomness.inBF[i])
		commitments.Inputs[i].Add(P)
		commitments.InputSum.Add(P)
	}
	// randomness used to prove sum value
	randomness.sum = p.Curve.NewRandomZr(rand)
	commitments.InputSum.Add(p.PedParams[1].Mul(randomness.sum))
	// add PedGen^{rand_type*len(p.Inputs)}
	commitments.InputSum.Add(Q.Mul(p.Curve.NewZrFromInt(int64(len(p.Inputs)))))

	// for outputs
	randomness.outValues = make([]*math.Zr, len(p.Outputs))
	randomness.outBF = make([]*math.Zr, len(p.Outputs))

	commitments.Outputs = make([]*math.G1, len(p.Outputs))
	commitments.OutputSum = p.Curve.NewG1()
	// randomness used to prove sum value
	commitments.OutputSum.Add(p.PedParams[1].Mul(randomness.sum))
	// add PedGen^{rand_type*len(p.Outputs)}
	commitments.OutputSum.Add(Q.Mul(p.Curve.NewZrFromInt(int64(len(p.Outputs)))))

	for i := 0; i < len(p.Outputs); i++ {
		// randomness to prove output values
		randomness.outValues[i] = p.Curve.NewRandomZr(rand)
		// randomness to prove output blinding factors
		randomness.outBF[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		commitments.Outputs[i] = p.PedParams[1].Mul(randomness.outValues[i])
		commitments.Outputs[i].Add(Q)
		P := p.PedParams[2].Mul(randomness.outBF[i])
		commitments.Outputs[i].Add(P)
		commitments.OutputSum.Add(P)
	}
	return commitments, randomness, nil
}

// GetInValues returns input values
func (w *WellFormednessWitness) GetInValues() []*math.Zr {
	return w.inValues
}

// GetOutValues returns output values
func (w *WellFormednessWitness) GetOutValues() []*math.Zr {
	return w.outValues
}

// GetOutBlindingFactors returns the randomness used in the Pedersen
// commitments in the outputs
func (w *WellFormednessWitness) GetOutBlindingFactors() []*math.Zr {
	return w.outBlindingFactors
}

// GetInBlindingFactors returns the randomness used in the Pedersen
// commitments in the inputs
func (w *WellFormednessWitness) GetInBlindingFactors() []*math.Zr {
	return w.inBlindingFactors
}
