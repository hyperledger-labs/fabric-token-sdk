/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

type TypeCorrectnessVerifier struct {
	PedersenParams []*bn256.G1
	TypeNym        *bn256.G1
	Token          *bn256.G1
	Message        []byte
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
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, errors.Errorf("failed to get random number generator")
	}
	// generate randomness

	randomness := &TypeCorrectnessRandomness{
		ttype:   bn256.RandModOrder(rand),
		sk:      bn256.RandModOrder(rand),
		tNymBF:  bn256.RandModOrder(rand),
		value:   bn256.RandModOrder(rand),
		tokenBF: bn256.RandModOrder(rand),
	}
	// compute commitments
	coms := &TypeCorrectnessCommitments{}
	coms.NYM, err = common.ComputePedersenCommitment([]*bn256.Zr{randomness.sk, randomness.ttype, randomness.tNymBF}, p.PedersenParams)
	if err != nil {
		return nil, errors.Errorf("failed to compute commitment")
	}
	coms.Token, err = common.ComputePedersenCommitment([]*bn256.Zr{randomness.ttype, randomness.value, randomness.tokenBF}, p.PedersenParams)
	if err != nil {
		return nil, errors.Errorf("failed to compute commitment")
	}
	// generate proof
	proof := &TypeCorrectness{}
	// compute challenge
	g1Array := common.GetG1Array([]*bn256.G1{p.TypeNym, p.Token, coms.NYM, coms.Token}, p.PedersenParams)
	bytes := g1Array.Bytes()
	bytes = append(bytes, p.Message...)
	proof.Challenge = bn256.HashModOrder(bytes)
	// compute proof
	proof.SK = bn256.ModAdd(bn256.ModMul(proof.Challenge, p.Witness.SK, bn256.Order), randomness.sk, bn256.Order)
	proof.Type = bn256.ModAdd(bn256.ModMul(proof.Challenge, p.Witness.Type, bn256.Order), randomness.ttype, bn256.Order)
	proof.TypeNymBF = bn256.ModAdd(bn256.ModMul(proof.Challenge, p.Witness.NymBF, bn256.Order), randomness.tNymBF, bn256.Order)
	proof.Value = bn256.ModAdd(bn256.ModMul(proof.Challenge, p.Witness.Value, bn256.Order), randomness.value, bn256.Order)
	proof.TokenBF = bn256.ModAdd(bn256.ModMul(proof.Challenge, p.Witness.TokenBF, bn256.Order), randomness.tokenBF, bn256.Order)

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
	coms.NYM, err = common.ComputePedersenCommitment([]*bn256.Zr{tc.SK, tc.Type, tc.TypeNymBF}, v.PedersenParams)
	if err != nil {
		return errors.Wrapf(err, "issuer verification has failed")
	}
	coms.NYM.Sub(v.TypeNym.Mul(tc.Challenge))

	coms.Token, err = common.ComputePedersenCommitment([]*bn256.Zr{tc.Type, tc.Value, tc.TokenBF}, v.PedersenParams)
	if err != nil {
		return errors.Wrapf(err, "issuer verification has failed")
	}
	coms.Token.Sub(v.Token.Mul(tc.Challenge))

	g1array := common.GetG1Array([]*bn256.G1{v.TypeNym, v.Token, coms.NYM, coms.Token}, v.PedersenParams)

	bytes := g1array.Bytes()
	bytes = append(bytes, v.Message...)
	// recompute challenge
	chal := bn256.HashModOrder(bytes)
	// check proof
	if chal.Cmp(tc.Challenge) != 0 {
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

func NewTypeCorrectnessProver(witness *TypeCorrectnessWitness, tnym, token *bn256.G1, message []byte, pp []*bn256.G1) *TypeCorrectnessProver {
	return &TypeCorrectnessProver{
		Witness:                 witness,
		TypeCorrectnessVerifier: NewTypeCorrectnessVerifier(tnym, token, message, pp),
	}
}

func NewTypeCorrectnessVerifier(tnym, token *bn256.G1, message []byte, pp []*bn256.G1) *TypeCorrectnessVerifier {
	return &TypeCorrectnessVerifier{
		PedersenParams: pp,
		TypeNym:        tnym,
		Token:          token,
		Message:        message,
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
