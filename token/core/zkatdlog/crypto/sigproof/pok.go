/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

// proof of knowledge of a Pointcheval-Sanders Signature
type POK struct {
	Challenge      *math.Zr
	Signature      *pssign.Signature // obfuscated signature
	Messages       []*math.Zr
	BlindingFactor *math.Zr
	Hash           *math.Zr
}

// witness for the proof of knowledge
type POKWitness struct {
	Messages       []*math.Zr
	Signature      *pssign.Signature
	BlindingFactor *math.Zr
}

type POKRandomness struct {
	messages       []*math.Zr
	hash           *math.Zr
	blindingFactor *math.Zr
}

type POKProver struct {
	*POKVerifier
	Witness    *POKWitness
	randomness *POKRandomness
}

type POKVerifier struct {
	PK    []*math.G2
	Q     *math.G2
	P     *math.G1
	Curve *math.Curve
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
	sprover := &common.SchnorrProver{Witness: common.GetZrArray(p.Witness.Messages, []*math.Zr{HashMessages(p.Witness.Messages, p.Curve), p.Witness.BlindingFactor}), Randomness: common.GetZrArray(p.randomness.messages, []*math.Zr{p.randomness.hash, p.randomness.blindingFactor}), Challenge: chal, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
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
func (p *POKProver) computeCommitment() (*math.Gt, error) {
	// Get RNG
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}
	// compute commitment
	p.randomness = &POKRandomness{}
	p.randomness.hash = p.Curve.NewRandomZr(rand)
	t := p.PK[len(p.Witness.Messages)+1].Mul(p.randomness.hash)
	p.randomness.messages = make([]*math.Zr, len(p.Witness.Messages))
	for i := 0; i < len(p.Witness.Messages); i++ {
		p.randomness.messages[i] = p.Curve.NewRandomZr(rand)
		t.Add(p.PK[i+1].Mul(p.randomness.messages[i]))
	}

	p.randomness.blindingFactor = p.Curve.NewRandomZr(rand)
	com := p.Curve.Pairing2(t, p.Witness.Signature.R, p.Q, p.P.Mul(p.randomness.blindingFactor))

	return p.Curve.FExp(com), nil
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
	if !proof.Challenge.Equals(chal) {
		return errors.Errorf("proof of PS signature is not valid")
	}
	return nil
}

func (v *POKVerifier) RecomputeCommitment(p *POK) (*math.Gt, error) {
	if len(v.PK) != len(p.Messages)+2 {
		return nil, errors.Errorf("length of signature public key does not match size of proof")
	}
	t := v.Curve.NewG2()
	for i := 0; i < len(p.Messages); i++ {
		t.Add(v.PK[i+1].Mul(p.Messages[i]))
	}
	t.Add(v.PK[len(p.Messages)+1].Mul(p.Hash))

	pk := v.Curve.NewG2()
	pk.Sub(v.PK[0])

	com := v.Curve.Pairing2(v.Q, p.Signature.S.Mul(p.Challenge), pk, p.Signature.R.Mul(p.Challenge))
	com.Inverse()
	com.Mul(v.Curve.Pairing2(t, p.Signature.R, v.Q, v.P.Mul(p.BlindingFactor)))

	return v.Curve.FExp(com), nil
}

func HashMessages(m []*math.Zr, c *math.Curve) *math.Zr {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return c.HashToZr(bytesToHash)
}

func (p *POKProver) obfuscateSignature() (*pssign.Signature, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	p.Witness.BlindingFactor = p.Curve.NewRandomZr(rand)
	v := pssign.NewVerifier(nil, nil, p.Curve)
	err = v.Randomize(p.Witness.Signature)
	if err != nil {
		return nil, err
	}
	sig := &pssign.Signature{}
	sig.Copy(p.Witness.Signature)
	sig.S.Add(p.P.Mul(p.Witness.BlindingFactor))

	return sig, nil
}

func (v *POKVerifier) computeChallenge(com *math.Gt, signature *pssign.Signature) (*math.Zr, error) {
	// serialize public inputs
	g2a := common.GetG2Array(v.PK, []*math.G2{v.Q})
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute challenge")
	}
	// compute challenge
	return v.Curve.HashToZr(common.GetBytesArray(v.P.Bytes(), g2a.Bytes(), bytes, com.Bytes())), nil

}
