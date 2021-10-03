/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
	bn256 "github.ibm.com/fabric-research/mathlib"
)

type TypeCorrectnessVerifier struct {
	PedersenParams []*bn256.G1
	TypeNym        *bn256.G1
	Token          *bn256.G1
	Message        []byte
	Curve          *bn256.Curve
}

type TypeCorrectnessProver struct {
	*TypeCorrectnessVerifier
	Witness *TypeCorrectnessWitness
}

type TypeCorrectness struct {
	SK        *bn256.Zr
	Type      *bn256.Zr
	TypeNymBF *bn256.Zr
	Value     *bn256.Zr
	TokenBF   *bn256.Zr
	Challenge *bn256.Zr
}

type TypeCorrectnessWitness struct {
	SK      *bn256.Zr
	Type    *bn256.Zr
	NymBF   *bn256.Zr
	Value   *bn256.Zr
	TokenBF *bn256.Zr
}

type TypeCorrectnessCommitments struct {
	NYM   *bn256.G1
	Token *bn256.G1
}

type TypeCorrectnessRandomness struct {
	sk      *bn256.Zr
	ttype   *bn256.Zr
	tNymBF  *bn256.Zr
	value   *bn256.Zr
	tokenBF *bn256.Zr
}

// prove that Authorization.Type and Authorization.Token encode the same type
func (p *TypeCorrectnessProver) Prove() ([]byte, error) {
	if len(p.PedersenParams) != 3 {
		return nil, errors.Errorf("provide Pedersen parameters of length 3")
	}
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get random number generator")
	}
	// generate randomness

	randomness := &TypeCorrectnessRandomness{
		ttype:   p.Curve.NewRandomZr(rand),
		sk:      p.Curve.NewRandomZr(rand),
		tNymBF:  p.Curve.NewRandomZr(rand),
		value:   p.Curve.NewRandomZr(rand),
		tokenBF: p.Curve.NewRandomZr(rand),
	}
	// compute commitments
	coms := &TypeCorrectnessCommitments{}
	coms.NYM, err = common.ComputePedersenCommitment([]*bn256.Zr{randomness.sk, randomness.ttype, randomness.tNymBF}, p.PedersenParams, p.Curve)
	if err != nil {
		return nil, errors.Errorf("failed to compute commitment")
	}
	coms.Token, err = common.ComputePedersenCommitment([]*bn256.Zr{randomness.ttype, randomness.value, randomness.tokenBF}, p.PedersenParams, p.Curve)
	if err != nil {
		return nil, errors.Errorf("failed to compute commitment")
	}
	// generate proof
	proof := &TypeCorrectness{}
	// compute challenge
	g1Array := common.GetG1Array([]*bn256.G1{p.TypeNym, p.Token, coms.NYM, coms.Token}, p.PedersenParams)
	bytes := g1Array.Bytes()
	bytes = append(bytes, p.Message...)
	proof.Challenge = p.Curve.HashToZr(bytes)
	// compute proof
	proof.SK = p.Curve.ModAdd(p.Curve.ModMul(proof.Challenge, p.Witness.SK, p.Curve.GroupOrder), randomness.sk, p.Curve.GroupOrder)
	proof.Type = p.Curve.ModAdd(p.Curve.ModMul(proof.Challenge, p.Witness.Type, p.Curve.GroupOrder), randomness.ttype, p.Curve.GroupOrder)
	proof.TypeNymBF = p.Curve.ModAdd(p.Curve.ModMul(proof.Challenge, p.Witness.NymBF, p.Curve.GroupOrder), randomness.tNymBF, p.Curve.GroupOrder)
	proof.Value = p.Curve.ModAdd(p.Curve.ModMul(proof.Challenge, p.Witness.Value, p.Curve.GroupOrder), randomness.value, p.Curve.GroupOrder)
	proof.TokenBF = p.Curve.ModAdd(p.Curve.ModMul(proof.Challenge, p.Witness.TokenBF, p.Curve.GroupOrder), randomness.tokenBF, p.Curve.GroupOrder)

	return proof.Serialize()
}

func (v *TypeCorrectnessVerifier) Verify(proof []byte) error {
	if len(v.PedersenParams) != 3 {
		return errors.Errorf("length of Pedersen parameters != 3")
	}
	tc := &TypeCorrectness{}
	err := tc.Deserialize(proof)
	if err != nil {
		return errors.Wrapf(err, "failed to parse issuer proof")
	}
	// recompute commitment from proof
	coms := TypeCorrectnessCommitments{}
	coms.NYM, err = common.ComputePedersenCommitment([]*bn256.Zr{tc.SK, tc.Type, tc.TypeNymBF}, v.PedersenParams, v.Curve)
	if err != nil {
		return errors.Wrapf(err, "issuer verification has failed")
	}
	coms.NYM.Sub(v.TypeNym.Mul(tc.Challenge))

	coms.Token, err = common.ComputePedersenCommitment([]*bn256.Zr{tc.Type, tc.Value, tc.TokenBF}, v.PedersenParams, v.Curve)
	if err != nil {
		return errors.Wrapf(err, "issuer verification has failed")
	}
	coms.Token.Sub(v.Token.Mul(tc.Challenge))

	g1array := common.GetG1Array([]*bn256.G1{v.TypeNym, v.Token, coms.NYM, coms.Token}, v.PedersenParams)

	bytes := g1array.Bytes()
	bytes = append(bytes, v.Message...)
	// recompute challenge
	chal := v.Curve.HashToZr(bytes)
	// check proof
	if !chal.Equals(tc.Challenge) {
		return errors.Errorf("origin of transaction is not authorized to issue")
	}

	return nil
}

func (p *TypeCorrectness) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *TypeCorrectness) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, &p)
}

func NewTypeCorrectnessProver(witness *TypeCorrectnessWitness, tnym, token *bn256.G1, message []byte, pp []*bn256.G1, curve *bn256.Curve) *TypeCorrectnessProver {
	return &TypeCorrectnessProver{
		Witness:                 witness,
		TypeCorrectnessVerifier: NewTypeCorrectnessVerifier(tnym, token, message, pp, curve),
	}
}

func NewTypeCorrectnessVerifier(tnym, token *bn256.G1, message []byte, pp []*bn256.G1, curve *bn256.Curve) *TypeCorrectnessVerifier {
	return &TypeCorrectnessVerifier{
		PedersenParams: pp,
		TypeNym:        tnym,
		Token:          token,
		Message:        message,
		Curve:          curve,
	}
}

func NewTypeCorrectnessWitness(sk, ttype, value, tNymBF, tokenBF *bn256.Zr) *TypeCorrectnessWitness {

	return &TypeCorrectnessWitness{
		SK:      sk,
		Type:    ttype,
		NymBF:   tNymBF,
		Value:   value,
		TokenBF: tokenBF,
	}
}
