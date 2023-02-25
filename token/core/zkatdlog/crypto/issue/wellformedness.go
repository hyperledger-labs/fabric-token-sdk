/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// WellFormedness shows that issued tokens contains Pedersen commitments to (type, value)
// WellFormedness also shows that all the issued tokens contain the same type
type WellFormedness struct {
	// Proof of type
	Type *math.Zr
	// Proof of values of tokens to be issued
	// i^th proof for the value of the i^th token
	Values []*math.Zr
	// Proof of randomness used to compute the commitment to type and value in the issued tokens
	// i^th proof is for the randomness  used to compute the i^th token
	BlindingFactors []*math.Zr
	// only when issue is not anonymous
	TypeInTheClear string
	// Challenge computed using the Fiat-Shamir Heuristic
	Challenge *math.Zr
}

// Serialize marshals WellFormedness proof
func (wf *WellFormedness) Serialize() ([]byte, error) {
	return json.Marshal(wf)
}

// Deserialize un-marshals WellFormedness proof
func (wf *WellFormedness) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, wf)
}

// WellFormednessRandomness is the randomness used to generate
// the well-formedness proof
type WellFormednessRandomness struct {
	blindingFactors []*math.Zr
	values          []*math.Zr
	ttype           *math.Zr
}

// WellFormednessProver contains information that allows an Issuer to prove that
// issued tokens are well formed
type WellFormednessProver struct {
	*WellFormednessVerifier
	// witness contains an array of TokenDataWitness
	// TokenDataWitness consists of (type, value, r) such that TokenData = g^type*h^value*f^r
	witness []*token.TokenDataWitness
	// randomness is the randomness during the proof generation
	randomness *WellFormednessRandomness
	// Commitments is the commitment to the randomness used to generate the proof
	Commitments []*math.G1
}

// NewWellFormednessProver returns a WellFormednessProver for the passed parameters
func NewWellFormednessProver(witness []*token.TokenDataWitness, tokens []*math.G1, anonymous bool, pp []*math.G1, c *math.Curve) *WellFormednessProver {
	return &WellFormednessProver{
		witness:                witness,
		WellFormednessVerifier: NewWellFormednessVerifier(tokens, anonymous, pp, c),
	}
}

// Prove returns a serialized wellformedness proof
func (p *WellFormednessProver) Prove() ([]byte, error) {
	if p.Curve == nil {
		return nil, errors.New("failed to generate well-formedness proof: please initialize curve")
	}
	err := p.computeCommitments()
	if err != nil {
		return nil, errors.Wrap(err, "The computation of the issue proof failed")
	}
	// compute challenge for proof
	raw, err := common.GetG1Array(p.Commitments, p.Tokens).Bytes()
	if err != nil {
		errors.Wrapf(err, "The computation of the issue proof failed")
	}
	// compute proof
	wf, err := p.computeProof(p.Curve.HashToZr(raw))
	if err != nil {
		return nil, errors.Wrap(err, "The computation of the issue proof failed")
	}
	// serialize proof
	return wf.Serialize()
}

// computeCommitments compute the commitments to the randomness used in the well-formedness proof
func (p *WellFormednessProver) computeCommitments() error {
	if len(p.PedParams) != 3 {
		return errors.New("computation of well-formedness proof failed: invalid public parameters")
	}
	if p.PedParams[0] == nil || p.PedParams[1] == nil || p.PedParams[2] == nil {
		return errors.New("computation of well-formedness proof failed: invalid public parameters")
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

// computeProof takes a challenge and returns a proof for
// the witnesses encoded in the WellFormednessProver
func (p *WellFormednessProver) computeProof(chal *math.Zr) (*WellFormedness, error) {
	if p.Curve.GroupOrder == nil {
		return nil, errors.New("computation of well-formedness proof failed: invalid public parameters")
	}
	wf := &WellFormedness{}
	// when issuer is hidden
	if p.Anonymous {
		// generate zkat proof for type of issued tokens
		if p.witness[0] == nil {
			return nil, errors.New("computation of well-formedness proof failed: invalid witness")
		}
		wf.Type = p.Curve.ModMul(chal, p.Curve.HashToZr([]byte(p.witness[0].Type)), p.Curve.GroupOrder)
		wf.Type = p.Curve.ModAdd(wf.Type, p.randomness.ttype, p.Curve.GroupOrder)
	} else {
		// if issue is not anonymous type is revealed
		wf.TypeInTheClear = p.witness[0].Type
	}

	var values []*math.Zr
	var bfs []*math.Zr
	for _, w := range p.witness {
		if w == nil {
			return nil, errors.New("computation of well-formedness proof failed: invalid witness")
		}
		values = append(values, w.Value)
		bfs = append(bfs, w.BlindingFactor)
	}

	// generate ZK proof for the values of the issued tokens
	sp := &common.SchnorrProver{Witness: values, Randomness: p.randomness.values, Challenge: chal, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	var err error
	wf.Values, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute well-formedness proof")
	}

	// generate ZK proof for the blindingFactors in the issued tokens
	sp = &common.SchnorrProver{Witness: bfs, Randomness: p.randomness.blindingFactors, Challenge: chal, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	wf.BlindingFactors, err = sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute well-formedness proof")
	}

	wf.Challenge = chal

	return wf, nil
}

// WellFormednessVerifier checks the validity of WellFormedness proof
type WellFormednessVerifier struct {
	PedParams []*math.G1
	Curve     *math.Curve
	Tokens    []*math.G1
	// Anonymous indicates if the issuance is anonymous
	Anonymous bool
}

// NewWellFormednessVerifier returns a WellFormednessVerifier corresponding to the passed parameters
func NewWellFormednessVerifier(tokens []*math.G1, anonymous bool, pp []*math.G1, c *math.Curve) *WellFormednessVerifier {
	return &WellFormednessVerifier{
		Tokens:    tokens,
		Anonymous: anonymous,
		PedParams: pp,
		Curve:     c,
	}
}

// Verify returns an error if the serialized proof is an invalid WellFormedness proof
func (v *WellFormednessVerifier) Verify(proof []byte) error {
	wf := &WellFormedness{}
	err := wf.Deserialize(proof)
	if err != nil {
		return errors.Wrap(err, "failed to verify well-formedness proof")
	}

	// parse ZK proofs
	zkps, err := v.parseProof(wf)
	if err != nil {
		return errors.Wrapf(err, "invalid zero-knowledge issue")
	}

	// recompute commitments used in ZK proofs
	// initialize scchnorr verifier
	ver := &common.SchnorrVerifier{PedParams: v.PedParams, Curve: v.Curve}
	coms, err := ver.RecomputeCommitments(zkps, wf.Challenge)
	if err != nil {
		return errors.Wrapf(err, "failed to verify well-formedness proof")
	}

	// recompute challenge and check proof validity
	raw, err := common.GetG1Array(coms, v.Tokens).Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to verify well-formedness proof")
	}
	if !v.Curve.HashToZr(raw).Equals(wf.Challenge) {
		return errors.Errorf("invalid well-formedness proof")
	}
	return nil
}

// parseProof takes a WellFormedness proof and returns the ZKPs that compose it
func (v *WellFormednessVerifier) parseProof(proof *WellFormedness) ([]*common.SchnorrProof, error) {
	if proof.Challenge == nil || v.Curve == nil || v.Curve.GroupOrder == nil {
		return nil, errors.New("failed to verify well-formedness proof: invalid public parameters")
	}
	if !v.Anonymous {
		proof.Type = v.Curve.ModMul(proof.Challenge, v.Curve.HashToZr([]byte(proof.TypeInTheClear)), v.Curve.GroupOrder)
	}
	if len(proof.Values) != len(v.Tokens) || len(proof.BlindingFactors) != len(v.Tokens) {
		return nil, errors.New("well-formedness proof is not well formed: length mismatch")
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
		if v.Tokens[i] == nil {
			return nil, errors.Errorf("well-formedness proof not well formed:  output token [%d] is nil", i)
		}
		zkps[i].Statement = v.Tokens[i].Copy()
	}
	return zkps, nil
}
