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

// TypeAndSumProof is zero-knowledge proof that shows that the inputs of a transaction
// have the same total value and the same type as its outputs
type TypeAndSumProof struct {
	// CommitmentToType is a pedersen commitment to the type of the inputs and the outputs
	CommitmentToType *math.G1
	// InputValues contains the proofs of knowledge of the values of the inputs
	InputValues []*math.Zr
	// OutputValues contains the proofs of knowledge of the values of the outputs
	OutputValues []*math.Zr
	// InputBlindingFactors contains the proofs of knowledge of the randomness
	// used in computing the inputs
	InputBlindingFactors []*math.Zr
	// OutputBlindingFactors contains the proofs of knowledge of the randomness
	// used in computing the outputs
	OutputBlindingFactors []*math.Zr
	// proof of knowledge of the token type encoded in both inputs and outputs
	Type *math.Zr
	// proof of knowledge of blinding factor used to compute the commitment to type
	TypeBlindingFactor *math.Zr
	// EqualityOfSum shows that the inputs and the outputs have the same value
	EqualityOfSum *math.Zr
	// Challenge is the challenge of the proof
	Challenge *math.Zr
}

// Serialize marshals TypeAndSumProof
func (p *TypeAndSumProof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize un-marshals TypeAndSumProof
func (p *TypeAndSumProof) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, p)
}

// TypeAndSumWitness contains the secret information used to produce TypeAndSumProof
type TypeAndSumWitness struct {
	// inValues carries the values of the inputs
	inValues []*math.Zr
	// outValues carries the values of the outputs
	outValues []*math.Zr
	// Type is the token type of inputs and outputs
	Type *math.Zr
	// inBlindingFactors carries the randomness used to compute the Pedersen commitments
	// in inputs
	inBlindingFactors []*math.Zr
	// outBlindingFactors carries the randomness used to compute the Pedersen commitments
	// in outputs
	outBlindingFactors []*math.Zr
	// typeBlindingFactor is the blinding factor used to compute the commitment to type
	typeBlindingFactor *math.Zr
}

// NewTypeAndSumWitness returns a TypeAndSumWitness as a function of the passed arguments
func NewTypeAndSumWitness(bf *math.Zr, in, out []*token.TokenDataWitness, c *math.Curve) *TypeAndSumWitness {
	inValues := make([]*math.Zr, len(in))
	outValues := make([]*math.Zr, len(out))
	inBF := make([]*math.Zr, len(in))
	outBF := make([]*math.Zr, len(out))
	for i := 0; i < len(in); i++ {
		inValues[i] = c.NewZrFromInt(int64(in[i].Value))
		inBF[i] = in[i].BlindingFactor
	}
	for i := 0; i < len(out); i++ {
		outValues[i] = c.NewZrFromInt(int64(out[i].Value))
		outBF[i] = out[i].BlindingFactor
	}
	return &TypeAndSumWitness{inValues: inValues, outValues: outValues, Type: c.HashToZr([]byte(in[0].Type)), inBlindingFactors: inBF, outBlindingFactors: outBF, typeBlindingFactor: bf}
}

// TypeAndSumProver produces a TypeAndSumProof proof
type TypeAndSumProver struct {
	// PedersenGenerators corresponds to the generators used to compute Pedersen commitments
	// (G_0, G_1, H)
	PedersenGenerators []*math.G1
	// Inputs are Pedersen commitments to (Type, Value) of the inputs to be spent
	Inputs []*math.G1
	// Outputs are Pedersen commitments to (Type, Value) of the outputs to be created
	// after the transfer
	Outputs []*math.G1
	// CommitmentToType is a Pedersen commitment to Type
	CommitmentToType *math.G1
	// witness is the secret information used to produce the proof
	witness *TypeAndSumWitness
	// Curve is the elliptic curve in which Pedersen commitments are computed
	Curve *math.Curve
}

// NewTypeAndSumProver returns a NewTypeAndSumProver as a function of the passed arguments
func NewTypeAndSumProver(witness *TypeAndSumWitness, pp []*math.G1, inputs []*math.G1, outputs []*math.G1, comType *math.G1, c *math.Curve) *TypeAndSumProver {
	return &TypeAndSumProver{witness: witness, CommitmentToType: comType, Inputs: inputs, Outputs: outputs, Curve: c, PedersenGenerators: pp}
}

// NewTypeAndSumVerifier returns a TypeAndSumVerifier as a function of the passed arguments
func NewTypeAndSumVerifier(pp []*math.G1, inputs []*math.G1, outputs []*math.G1, c *math.Curve) *TypeAndSumVerifier {
	return &TypeAndSumVerifier{Inputs: inputs, Outputs: outputs, PedersenGenerators: pp, Curve: c}
}

// TypeAndSumVerifier checks the validity of TypeAndSumProof
type TypeAndSumVerifier struct {
	// PedersenGenerators corresponds to the generators used to compute Pedersen commitments
	// (g_1, g_2, h)
	PedersenGenerators []*math.G1
	// Curve is the elliptic curve in which Pedersen commitments are computed
	Curve *math.Curve
	// Inputs are Pedersen commitments to (Type, Value) of the inputs to be spent
	Inputs []*math.G1
	// Outputs are Pedersen commitments to (Type, Value) of the outputs to be created
	// after the transfer
	Outputs []*math.G1
}

// TypeAndSumProofRandomness is the randomness used in the generation of TypeAndSumProof
type TypeAndSumProofRandomness struct {
	inValues           []*math.Zr
	inBlindingFactor   []*math.Zr
	outValues          []*math.Zr
	outBlindingFactor  []*math.Zr
	tokenType          *math.Zr
	typeBlindingFactor *math.Zr
	sumBlindingFactor  *math.Zr
}

// TypeAndSumProofCommitments are commitments to the randomness used in TypeAndSumProof
type TypeAndSumProofCommitments struct {
	Inputs           []*math.G1
	Outputs          []*math.G1
	Sum              *math.G1
	CommitmentToType *math.G1
}

// Prove returns a TypeAndSumProof. Prove computes a Pedersen commitment to type T
// Given T, the prover computes for each input "IN" and output "OUT", Prove computes
// IN/T and OUT/T, and shows IN/T and OUT/T are commitments of the form G_1^vH^r
// Prove then proves that (\prod IN/T)/(\prod Out/T) is a commitment of the form H^s
func (p *TypeAndSumProver) Prove() (*TypeAndSumProof, error) {
	// generate randomness for the proof and compute the corresponding commitments
	commitments, randomness, err := p.computeCommitments()
	if err != nil {
		return nil, err
	}
	var inputs, outputs []*math.G1
	sum := p.Curve.NewG1() // sum = \prod (inputs[i]/commitmentToType)/ \prod (outputs[i]/commitmentToType)
	for i := 0; i < len(p.Inputs); i++ {
		// compute in = inputs[i]/commitmentToType
		in := p.Inputs[i].Copy()
		in.Sub(p.CommitmentToType)
		inputs = append(inputs, in)
		sum.Add(in)
	}
	for i := 0; i < len(p.Outputs); i++ {
		// compute out = outputs[i]/commitmentToType
		out := p.Outputs[i].Copy()
		out.Sub(p.CommitmentToType)
		outputs = append(outputs, out)
		sum.Sub(out)
	}

	// serialize public information
	raw, err := crypto.GetG1Array(commitments.Inputs, commitments.Outputs, []*math.G1{commitments.CommitmentToType, commitments.Sum}, inputs, outputs, []*math.G1{p.CommitmentToType, sum}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "cannot compute sum and type proof")
	}

	// compute proofs
	// prove that inputs and outputs are of the form G_1^vH^r
	// prove that \prod inputs[i] / \prod outputs[i] is of the form H^s
	stp, err := p.computeProof(randomness, p.Curve.HashToZr(raw))
	if err != nil {
		return nil, err
	}

	return stp, nil
}

// Verify returns an error when TypeAndSumProof is not a valid
func (v *TypeAndSumVerifier) Verify(stp *TypeAndSumProof) error {

	if stp.TypeBlindingFactor == nil || stp.Type == nil || stp.CommitmentToType == nil || stp.EqualityOfSum == nil {
		return errors.New("invalid sum and type proof")
	}

	var inputs, outputs []*math.G1
	sum := v.Curve.NewG1()
	var inComs, outComs []*math.G1

	// verify that inputs[i]/stp.CommitmentToType is of the form G_1^vH^r
	// verify that outputs[i]/stp.commitmentToType is of the form G_1^vH^r
	// verify that stp.CommitmentToType is of the form G_0^typeH^r
	// verify that (inputs[i]/stp.CommitmentToType) \ (\prod outputs[i]/stp.commitmentToType)
	// is of the form H^s

	for i := 0; i < len(v.Inputs); i++ {
		if stp.InputValues[i] == nil {
			return errors.New("invalid sum and type proof")
		}
		in := v.Inputs[i].Copy()
		in.Sub(stp.CommitmentToType)
		inputs = append(inputs, in)

		sum.Add(in)

		inC := v.PedersenGenerators[1].Mul(stp.InputValues[i])
		inC.Add(v.PedersenGenerators[2].Mul(stp.InputBlindingFactors[i]))
		inC.Sub(in.Mul(stp.Challenge))
		inComs = append(inComs, inC)
	}

	// todo : proof for outputs is not needed. It is taken care by the range proof.
	for i := 0; i < len(v.Outputs); i++ {
		if stp.OutputValues[i] == nil {
			return errors.New("invalid sum and type proof")
		}
		out := v.Outputs[i].Copy()
		out.Sub(stp.CommitmentToType)
		outputs = append(outputs, out)

		sum.Sub(out)

		outC := v.PedersenGenerators[1].Mul(stp.OutputValues[i])
		outC.Add(v.PedersenGenerators[2].Mul(stp.OutputBlindingFactors[i]))
		outC.Sub(out.Mul(stp.Challenge))
		outComs = append(outComs, outC)
	}
	sumCom := v.PedersenGenerators[2].Mul(stp.EqualityOfSum)
	sumCom.Sub(sum.Mul(stp.Challenge))

	typeCom := v.PedersenGenerators[0].Mul(stp.Type)
	typeCom.Add(v.PedersenGenerators[2].Mul(stp.TypeBlindingFactor))
	typeCom.Sub(stp.CommitmentToType.Mul(stp.Challenge))

	raw, err := crypto.GetG1Array(inComs, outComs, []*math.G1{typeCom, sumCom}, inputs, outputs, []*math.G1{stp.CommitmentToType, sum}).Bytes()
	if err != nil {
		return errors.Wrap(err, "cannot verify sum and type proof")
	}
	// compute challenge
	chal := v.Curve.HashToZr(raw)
	if !chal.Equals(stp.Challenge) {
		return errors.New("invalid sum and type proof")
	}
	return nil
}

// computeProof compute the proof as a function of the passed randomness and challenge
func (p *TypeAndSumProver) computeProof(randomness *TypeAndSumProofRandomness, chal *math.Zr) (*TypeAndSumProof, error) {
	stp := &TypeAndSumProof{CommitmentToType: p.CommitmentToType, Challenge: chal}

	// generate zk proof for commitment to type
	stp.Type = p.Curve.ModMul(chal, p.witness.Type, p.Curve.GroupOrder)
	stp.Type = p.Curve.ModAdd(stp.Type, randomness.tokenType, p.Curve.GroupOrder)

	stp.TypeBlindingFactor = p.Curve.ModMul(chal, p.witness.typeBlindingFactor, p.Curve.GroupOrder)
	stp.TypeBlindingFactor = p.Curve.ModAdd(stp.TypeBlindingFactor, randomness.typeBlindingFactor, p.Curve.GroupOrder)

	sumBF := p.Curve.NewZrFromInt(0)
	// generate zk proof for input values and corresponding blinding factors
	for i := 0; i < len(p.Inputs); i++ {
		v := p.Curve.ModMul(chal, p.witness.inValues[i], p.Curve.GroupOrder)
		v = p.Curve.ModAdd(v, randomness.inValues[i], p.Curve.GroupOrder)
		stp.InputValues = append(stp.InputValues, v)

		t := p.Curve.ModSub(p.witness.inBlindingFactors[i], p.witness.typeBlindingFactor, p.Curve.GroupOrder)
		bf := p.Curve.ModMul(chal, t, p.Curve.GroupOrder)
		bf = p.Curve.ModAdd(bf, randomness.inBlindingFactor[i], p.Curve.GroupOrder)
		stp.InputBlindingFactors = append(stp.InputBlindingFactors, bf)
		sumBF = p.Curve.ModAdd(sumBF, t, p.Curve.GroupOrder)
	}

	// generate zk proof for output values and corresponding blinding factors
	for i := 0; i < len(p.Outputs); i++ {
		v := p.Curve.ModMul(chal, p.witness.outValues[i], p.Curve.GroupOrder)
		v = p.Curve.ModAdd(v, randomness.outValues[i], p.Curve.GroupOrder)
		stp.OutputValues = append(stp.OutputValues, v)

		t := p.Curve.ModSub(p.witness.outBlindingFactors[i], p.witness.typeBlindingFactor, p.Curve.GroupOrder)
		bf := p.Curve.ModMul(chal, t, p.Curve.GroupOrder)
		bf = p.Curve.ModAdd(bf, randomness.outBlindingFactor[i], p.Curve.GroupOrder)
		stp.OutputBlindingFactors = append(stp.OutputBlindingFactors, bf)

		sumBF = p.Curve.ModSub(sumBF, t, p.Curve.GroupOrder)
	}

	// generate zk to show equality of sum
	stp.EqualityOfSum = p.Curve.ModMul(chal, sumBF, p.Curve.GroupOrder)
	stp.EqualityOfSum = p.Curve.ModAdd(stp.EqualityOfSum, randomness.sumBlindingFactor, p.Curve.GroupOrder)

	return stp, nil
}

// computeCommitments returns the randomness used in TypeAndSumProof proof and the corresponding commitments
func (p *TypeAndSumProver) computeCommitments() (*TypeAndSumProofCommitments, *TypeAndSumProofRandomness, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, err
	}

	commitments := &TypeAndSumProofCommitments{}
	// produce randomness for the TypeAndSumProof proof
	randomness := &TypeAndSumProofRandomness{}

	// for commitment to type
	randomness.tokenType = p.Curve.NewRandomZr(rand) // randomness to prove token type
	randomness.typeBlindingFactor = p.Curve.NewRandomZr(rand)

	commitments.CommitmentToType = p.PedersenGenerators[0].Mul(randomness.tokenType)
	commitments.CommitmentToType.Add(p.PedersenGenerators[2].Mul(randomness.typeBlindingFactor))

	// for inputs
	randomness.inValues = make([]*math.Zr, len(p.Inputs))
	randomness.inBlindingFactor = make([]*math.Zr, len(p.Inputs))
	commitments.Inputs = make([]*math.G1, len(p.Inputs))

	for i := 0; i < len(p.Inputs); i++ {
		// randomness to prove input values
		randomness.inValues[i] = p.Curve.NewRandomZr(rand)
		// randomness to prove input blinding factors
		randomness.inBlindingFactor[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		commitments.Inputs[i] = p.PedersenGenerators[1].Mul(randomness.inValues[i])
		commitments.Inputs[i].Add(p.PedersenGenerators[2].Mul(randomness.inBlindingFactor[i]))
	}
	// for outputs
	randomness.outValues = make([]*math.Zr, len(p.Outputs))
	randomness.outBlindingFactor = make([]*math.Zr, len(p.Outputs))
	commitments.Outputs = make([]*math.G1, len(p.Outputs))

	for i := 0; i < len(p.Outputs); i++ {
		// randomness to prove output values
		randomness.outValues[i] = p.Curve.NewRandomZr(rand)
		// randomness to prove output blinding factors
		randomness.outBlindingFactor[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		commitments.Outputs[i] = p.PedersenGenerators[1].Mul(randomness.outValues[i])
		commitments.Outputs[i].Add(p.PedersenGenerators[2].Mul(randomness.outBlindingFactor[i]))
	}

	// for sum
	randomness.sumBlindingFactor = p.Curve.NewRandomZr(rand)
	commitments.Sum = p.PedersenGenerators[2].Mul(randomness.sumBlindingFactor)

	return commitments, randomness, nil
}
