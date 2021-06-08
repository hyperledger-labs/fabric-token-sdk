/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// proof of knowledge of a Pointcheval-Sanders Signature
type POK struct {
	Challenge      *bn256.Zr
	Signature      *pssign.Signature // obfuscated signature
	Messages       []*bn256.Zr
	BlindingFactor *bn256.Zr
	Hash           *bn256.Zr
}

// witness for the proof of knowledge
type POKWitness struct {
	Messages       []*bn256.Zr
	Signature      *pssign.Signature
	BlindingFactor *bn256.Zr
}

type POKRandomness struct {
	messages       []*bn256.Zr
	hash           *bn256.Zr
	blindingFactor *bn256.Zr
}

type POKProver struct {
	*POKVerifier
	Witness    *POKWitness
	randomness *POKRandomness
}

type POKVerifier struct {
	PK []*bn256.G2
	Q  *bn256.G2
	P  *bn256.G1
}

func (p *POKProver) Prove() ([]byte, error) {
	// randomize signature
	sig, err := p.obfuscateSignature()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate proof of knowledge of PS signature")
	}
	// compute commitment
	com, err := p.computeCommitment()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute proof of knowledge of PS signature")
	}

	chal, err := p.computeChallenge(com, sig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate proof of knowledge of PS signature")
	}
	// generate schnorr proof
	sprover := &common.SchnorrProver{Witness: common.GetZrArray(p.Witness.Messages, []*bn256.Zr{HashMessages(p.Witness.Messages), p.Witness.BlindingFactor}), Randomness: common.GetZrArray(p.randomness.messages, []*bn256.Zr{p.randomness.hash, p.randomness.blindingFactor}), Challenge: chal}
	sp, err := sprover.Prove()
	if err != nil {
		return nil, errors.Errorf("failed to compute proof of knowledge of PS signature")
	}

	// serialize proof for PS signature
	return json.Marshal(
		&POK{
			Challenge:      chal,
			Signature:      sig,
			Messages:       sp[:len(p.Witness.Messages)],
			Hash:           sp[len(p.Witness.Messages)],
			BlindingFactor: sp[len(p.Witness.Messages)+1]},
	)
}
func (p *POKProver) computeCommitment() (*bn256.GT, error) {
	// Get RNG
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}
	// compute commitment
	p.randomness = &POKRandomness{}
	p.randomness.hash = bn256.RandModOrder(rand)
	t := p.PK[len(p.Witness.Messages)+1].Mul(p.randomness.hash)
	p.randomness.messages = make([]*bn256.Zr, len(p.Witness.Messages))
	for i := 0; i < len(p.Witness.Messages); i++ {
		p.randomness.messages[i] = bn256.RandModOrder(rand)
		t.Add(p.PK[i+1].Mul(p.randomness.messages[i]))
	}

	p.randomness.blindingFactor = bn256.RandModOrder(rand)
	com := bn256.Pairing(t, p.Witness.Signature.R, p.Q, p.P.Mul(p.randomness.blindingFactor))

	return bn256.FinalExp(com), nil
}

func (v *POKVerifier) Verify(p []byte) error {
	proof := &POK{}
	err := json.Unmarshal(p, proof)
	if err != nil {
		return errors.Wrapf(err, "failed to verify POK of PS signature")
	}
	// get commitment bytes
	com, err := v.RecomputeCommitment(proof)
	if err != nil {
		return errors.Wrapf(err, "failed to verify POK of PS signature")
	}

	// recompute challenge
	chal, err := v.computeChallenge(com, proof.Signature)
	if err != nil {
		return errors.Wrapf(err, "failed to verify POK of PS signature")
	}

	// check proof is valid
	if proof.Challenge.Cmp(chal) != 0 {
		return errors.Errorf("proof of PS signature is not valid")
	}
	return nil
}

func (v *POKVerifier) RecomputeCommitment(p *POK) (*bn256.GT, error) {
	if len(v.PK) != len(p.Messages)+2 {
		return nil, errors.Errorf("length of signature public key does not match size of proof")
	}
	t := bn256.NewG2()
	for i := 0; i < len(p.Messages); i++ {
		t.Add(v.PK[i+1].Mul(p.Messages[i]))
	}
	t.Add(v.PK[len(p.Messages)+1].Mul(p.Hash))

	pk := bn256.NewG2()
	pk.Sub(v.PK[0])

	com := bn256.Pairing(v.Q, p.Signature.S.Mul(p.Challenge), pk, p.Signature.R.Mul(p.Challenge))
	com.Inverse()
	com.Mul(bn256.Pairing(t, p.Signature.R, v.Q, v.P.Mul(p.BlindingFactor)))

	return bn256.FinalExp(com), nil
}

func HashMessages(m []*bn256.Zr) *bn256.Zr {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return bn256.HashModOrder(bytesToHash)
}

func (p *POKProver) obfuscateSignature() (*pssign.Signature, error) {
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	p.Witness.BlindingFactor = bn256.RandModOrder(rand)
	err = p.Witness.Signature.Randomize()
	if err != nil {
		return nil, err
	}
	sig := &pssign.Signature{}
	sig.Copy(p.Witness.Signature)
	sig.S.Add(p.P.Mul(p.Witness.BlindingFactor))

	return sig, nil
}

func (v *POKVerifier) computeChallenge(com *bn256.GT, signature *pssign.Signature) (*bn256.Zr, error) {
	// serialize public inputs
	g2a := common.GetG2Array(v.PK, []*bn256.G2{v.Q})
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute challenge")
	}
	// compute challenge
	return bn256.HashModOrder(common.GetBytesArray(v.P.Bytes(), g2a.Bytes(), bytes, com.Bytes())), nil

}
