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
	// a pedersen commitment to the type of the inputs and the outputs
	CommitmentToType *math.G1
	// proof of knowledge of the randomness used in Pedersen commitments in the inputs
	InputBlindingFactors []*math.Zr
	// proof of knowledge of the values encoded in the Pedersen commitments in the inputs
	InputValues []*math.Zr
	// proof of knowledge of the token type encoded in both inputs and outputs
	Type *math.Zr
	// proof of knowledge of blinding factor used to compute the commitment to type
	TypeBlindingFactor *math.Zr
	// proof of knowledge of equality of sum
	EqualityOfSum *math.Zr
	// challenge used in proof
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
	// PedParams corresponds to the generators used to compute Pedersen commitments
	// (g_1, g_2, h)
	PedParams []*math.G1
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
	return &TypeAndSumProver{witness: witness, CommitmentToType: comType, Inputs: inputs, Outputs: outputs, Curve: c, PedParams: pp}
}

// NewTypeAndSumVerifier returns a TypeAndSumVerifier as a function of the passed arguments
func NewTypeAndSumVerifier(pp []*math.G1, inputs []*math.G1, outputs []*math.G1, c *math.Curve) *TypeAndSumVerifier {
	return &TypeAndSumVerifier{Inputs: inputs, Outputs: outputs, PedParams: pp, Curve: c}
}

// TypeAndSumVerifier checks the validity of TypeAndSumProof
type TypeAndSumVerifier struct {
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

// TypeAndSumProofRandomness is the randomness used in the generation of TypeAndSumProof
type TypeAndSumProofRandomness struct {
	inValues []*math.Zr
	inBF     []*math.Zr
	ttype    *math.Zr
	typeBF   *math.Zr
	sumBF    *math.Zr
}

// TypeAndSumProofCommitments are commitments to the randomness used in TypeAndSumProof
type TypeAndSumProofCommitments struct {
	Inputs           []*math.G1
	Sum              *math.G1
	CommitmentToType *math.G1
}

// Prove returns a serialized TypeAndSumProof proof
func (p *TypeAndSumProver) Prove() (*TypeAndSumProof, error) {
	// generate randomness for the proof and compute the corresponding commitments
	commitments, randomness, err := p.computeCommitments()
	if err != nil {
		return nil, err
	}
	var inputs, outputs []*math.G1
	// sum = \prod (inputs[i]/commitmentToType)/ \prod (outputs[i]/commitmentToType)
	// sum = G_2^r
	sum := p.Curve.NewG1()
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
	raw, err := crypto.GetG1Array(commitments.Inputs, []*math.G1{commitments.CommitmentToType, commitments.Sum}, inputs, outputs, []*math.G1{p.CommitmentToType, sum}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "cannot compute sum and type proof")
	}
	// compute challenge
	chal := p.Curve.HashToZr(raw)
	// compute proofs
	stp, err := p.computeProof(randomness, chal)
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

	var inComs []*math.G1

	for i := 0; i < len(v.Inputs); i++ {
		if stp.InputValues[i] == nil {
			return errors.New("invalid sum and type proof")
		}
		in := v.Inputs[i].Copy()
		in.Sub(stp.CommitmentToType)
		inputs = append(inputs, in)

		sum.Add(in)

		inC := v.PedParams[1].Mul(stp.InputValues[i])
		inC.Add(v.PedParams[2].Mul(stp.InputBlindingFactors[i]))
		inC.Sub(in.Mul(stp.Challenge))
		inComs = append(inComs, inC)
	}

	for i := 0; i < len(v.Outputs); i++ {
		out := v.Outputs[i].Copy()
		out.Sub(stp.CommitmentToType)
		outputs = append(outputs, out)

		sum.Sub(out)

	}
	sumCom := v.PedParams[2].Mul(stp.EqualityOfSum)
	sumCom.Sub(sum.Mul(stp.Challenge))

	typeCom := v.PedParams[0].Mul(stp.Type)
	typeCom.Add(v.PedParams[2].Mul(stp.TypeBlindingFactor))
	typeCom.Sub(stp.CommitmentToType.Mul(stp.Challenge))

	raw, err := crypto.GetG1Array(inComs, []*math.G1{typeCom, sumCom}, inputs, outputs, []*math.G1{stp.CommitmentToType, sum}).Bytes()
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
	stp.Type = p.Curve.ModAdd(stp.Type, randomness.ttype, p.Curve.GroupOrder)

	stp.TypeBlindingFactor = p.Curve.ModMul(chal, p.witness.typeBlindingFactor, p.Curve.GroupOrder)
	stp.TypeBlindingFactor = p.Curve.ModAdd(stp.TypeBlindingFactor, randomness.typeBF, p.Curve.GroupOrder)

	sumBF := p.Curve.NewZrFromInt(0)
	// generate zk proof for input values and corresponding blinding factors
	for i := 0; i < len(p.Inputs); i++ {
		v := p.Curve.ModMul(chal, p.witness.inValues[i], p.Curve.GroupOrder)
		v = p.Curve.ModAdd(v, randomness.inValues[i], p.Curve.GroupOrder)
		stp.InputValues = append(stp.InputValues, v)

		t := p.Curve.ModSub(p.witness.inBlindingFactors[i], p.witness.typeBlindingFactor, p.Curve.GroupOrder)
		bf := p.Curve.ModMul(chal, t, p.Curve.GroupOrder)
		bf = p.Curve.ModAdd(bf, randomness.inBF[i], p.Curve.GroupOrder)
		stp.InputBlindingFactors = append(stp.InputBlindingFactors, bf)
		sumBF = p.Curve.ModAdd(sumBF, t, p.Curve.GroupOrder)
	}

	// we don't generate proof that outputs[i]/commitmentToType = G_1^vG_2^r as this is taken care of by
	// range proofs
	for i := 0; i < len(p.Outputs); i++ {
		t := p.Curve.ModSub(p.witness.outBlindingFactors[i], p.witness.typeBlindingFactor, p.Curve.GroupOrder)
		sumBF = p.Curve.ModSub(sumBF, t, p.Curve.GroupOrder)
	}

	// generate zk to show equality of sum
	stp.EqualityOfSum = p.Curve.ModMul(chal, sumBF, p.Curve.GroupOrder)
	stp.EqualityOfSum = p.Curve.ModAdd(stp.EqualityOfSum, randomness.sumBF, p.Curve.GroupOrder)

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
	randomness.ttype = p.Curve.NewRandomZr(rand) // randomness to prove token type
	randomness.typeBF = p.Curve.NewRandomZr(rand)

	commitments.CommitmentToType = p.PedParams[0].Mul(randomness.ttype)
	commitments.CommitmentToType.Add(p.PedParams[2].Mul(randomness.typeBF))

	// for inputs
	randomness.inValues = make([]*math.Zr, len(p.Inputs))
	randomness.inBF = make([]*math.Zr, len(p.Inputs))
	commitments.Inputs = make([]*math.G1, len(p.Inputs))

	for i := 0; i < len(p.Inputs); i++ {
		// randomness to prove input values
		randomness.inValues[i] = p.Curve.NewRandomZr(rand)
		// randomness to prove input blinding factors
		randomness.inBF[i] = p.Curve.NewRandomZr(rand)
		// compute corresponding commitments
		commitments.Inputs[i] = p.PedParams[1].Mul(randomness.inValues[i])
		commitments.Inputs[i].Add(p.PedParams[2].Mul(randomness.inBF[i]))
	}

	// for sum
	randomness.sumBF = p.Curve.NewRandomZr(rand)
	commitments.Sum = p.PedParams[2].Mul(randomness.sumBF)

	return commitments, randomness, nil
}
