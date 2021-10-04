/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// issued token correctness proof (WellFormedness)
type WellFormedness struct {
	Type            *math.Zr
	Values          []*math.Zr
	BlindingFactors []*math.Zr
	TypeInTheClear  string // only when issue is not anonymous
	Challenge       *math.Zr
}

// witness for issue
func NewTokenDataWitness(ttype string, values, bfs []*math.Zr) []*token.TokenDataWitness {
	witness := make([]*token.TokenDataWitness, len(values))
	for i, v := range values {
		witness[i] = &token.TokenDataWitness{Value: v, BlindingFactor: bfs[i]}
	}
	witness[0].Type = ttype
	return witness
}

func (wf *WellFormedness) Serialize() ([]byte, error) {
	return json.Marshal(wf)
}

func (wf *WellFormedness) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, wf)
}

// randomness used in well-formedness proof
type WellFormednessRandomness struct {
	blindingFactors []*math.Zr
	values          []*math.Zr
	ttype           *math.Zr
}

// zero knowledge verifier for issue
type WellFormednessVerifier struct {
	*common.SchnorrVerifier
	Tokens    []*math.G1
	Anonymous bool
}

type WellFormednessProver struct {
	*WellFormednessVerifier
	witness     []*token.TokenDataWitness
	randomness  *WellFormednessRandomness
	Commitments []*math.G1
}

func NewWellFormednessProver(witness []*token.TokenDataWitness, tokens []*math.G1, anonymous bool, pp []*math.G1, c *math.Curve) *WellFormednessProver {
	return &WellFormednessProver{
		witness:                witness,
		WellFormednessVerifier: NewWellFormednessVerifier(tokens, anonymous, pp, c),
	}
}

func NewWellFormednessVerifier(tokens []*math.G1, anonymous bool, pp []*math.G1, c *math.Curve) *WellFormednessVerifier {
	return &WellFormednessVerifier{
		Tokens:          tokens,
		Anonymous:       anonymous,
		SchnorrVerifier: &common.SchnorrVerifier{PedParams: pp, Curve: c},
	}
}

func (p *WellFormednessProver) Prove() ([]byte, error) {
	err := p.computeCommitments()
	if err != nil {
		return nil, errors.Wrap(err, "The computation of the transfer proof failed 1")
	}
	// compute challenge for proof
	chal := p.SchnorrVerifier.ComputeChallenge(common.GetG1Array(p.Commitments, p.Tokens))
	// compute proof
	wf, err := p.computeProof(chal)
	if err != nil {
		return nil, errors.Wrap(err, "The computation of the transfer proof failed 3")
	}
	// serialize proof
	return wf.Serialize()
}

func (v *WellFormednessVerifier) Verify(proof []byte) error {
	wf := &WellFormedness{}
	err := wf.Deserialize(proof)
	if err != nil {
		return err
	}
	// initialize scchnorr verifier
	ver := &common.SchnorrVerifier{PedParams: v.PedParams, Curve: v.Curve}
	// parse proof
	zkps := v.parseProof(wf)
	// recompute commitments used in proof
	coms := ver.RecomputeCommitments(zkps, wf.Challenge)
	// recompute challenge
	chal := v.SchnorrVerifier.ComputeChallenge(common.GetG1Array(coms, v.Tokens))
	// check proof
	if !chal.Equals(wf.Challenge) {
		return errors.Errorf("invalid zero-knowledge issue")
	}
	return nil
}

func (v *WellFormednessVerifier) parseProof(proof *WellFormedness) []*common.SchnorrProof {
	if !v.Anonymous {
		proof.Type = v.Curve.ModMul(proof.Challenge, v.Curve.HashToZr([]byte(proof.TypeInTheClear)), v.Curve.GroupOrder)
	}
	// parse proof
	zkps := make([]*common.SchnorrProof, len(v.Tokens))
	for i := 0; i < len(zkps); i++ {
		zkps[i] = &common.SchnorrProof{}
		zkps[i].Proof = make([]*math.Zr, 3)
		zkps[i].Proof[0] = proof.Type
		zkps[i].Proof[1] = proof.Values[i]
		zkps[i].Proof[2] = proof.BlindingFactors[i]
		zkps[i].Statement = v.Curve.NewG1()
		zkps[i].Statement = v.Tokens[i].Copy()
	}

	return zkps
}

func (p *WellFormednessProver) computeCommitments() error {
	if len(p.PedParams) != 3 {
		return errors.Errorf("computation of issue proof failed: invalid public parameters")
	}
	// get random number generator
	rand, err := p.Curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	// randomness for proof
	p.randomness = &WellFormednessRandomness{}
	p.Commitments = make([]*math.G1, len(p.Tokens))
	p.randomness.values = make([]*math.Zr, len(p.Tokens))
	p.randomness.blindingFactors = make([]*math.Zr, len(p.Tokens))

	Q := p.Curve.NewG1()
	// if issuer is hidden compute commitment for type randomness
	if p.Anonymous {
		// randomness for type proof
		p.randomness.ttype = p.Curve.NewRandomZr(rand)
		Q = p.PedParams[0].Mul(p.randomness.ttype)
	}
	// compute commitment
	for i := 0; i < len(p.Tokens); i++ {
		// randomness for value proof
		p.randomness.values[i] = p.Curve.NewRandomZr(rand)
		p.Commitments[i] = p.PedParams[1].Mul(p.randomness.values[i])
		// randomness for blinding factor proof
		p.randomness.blindingFactors[i] = p.Curve.NewRandomZr(rand)
		p.Commitments[i].Add(p.PedParams[2].Mul(p.randomness.blindingFactors[i]))
		// add type
		p.Commitments[i].Add(Q)
	}
	return nil
}

func (p *WellFormednessProver) computeProof(chal *math.Zr) (*WellFormedness, error) {

	wf := &WellFormedness{}
	// when issuer is hidden
	if p.Anonymous {
		// generate zkat proof for type of issued tokens
		wf.Type = p.Curve.ModMul(chal, p.Curve.HashToZr([]byte(p.witness[0].Type)), p.Curve.GroupOrder)
		wf.Type = p.Curve.ModAdd(wf.Type, p.randomness.ttype, p.Curve.GroupOrder)
	} else {
		// if issue is not anonymous type is revealed
		wf.TypeInTheClear = p.witness[0].Type
	}

	var values []*math.Zr
	var bfs []*math.Zr
	for _, w := range p.witness {
		values = append(values, w.Value)
		bfs = append(bfs, w.BlindingFactor)
	}

	// generate zkat proof for the values of the issued tokens
	sp := &common.SchnorrProver{Witness: values, Randomness: p.randomness.values, Challenge: chal, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	var err error
	wf.Values, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the issued token values")
	}

	// generate zkat proof for the blindingFactors in the issued tokens
	sp = &common.SchnorrProver{Witness: bfs, Randomness: p.randomness.blindingFactors, Challenge: chal, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	wf.BlindingFactors, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof for the blindingFactors in the issued tokens")
	}

	wf.Challenge = chal

	return wf, nil
}
