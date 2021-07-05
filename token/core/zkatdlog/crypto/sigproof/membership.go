/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

// membership proof based on ps signature
type MembershipProof struct {
	Challenge         *bn256.Zr
	Signature         *pssign.Signature
	Value             *bn256.Zr
	ComBlindingFactor *bn256.Zr
	SigBlindingFactor *bn256.Zr
	Hash              *bn256.Zr
	Commitment        *bn256.G1
}

func (p *MembershipProof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *MembershipProof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

// witness for membership proof
type MembershipWitness struct {
	signature         *pssign.Signature
	value             *bn256.Zr
	hash              *bn256.Zr
	sigBlindingFactor *bn256.Zr
	comBlidingFactor  *bn256.Zr
}

func NewMembershipWitness(sig *pssign.Signature, value *bn256.Zr, bf *bn256.Zr) *MembershipWitness {
	return &MembershipWitness{signature: sig, value: value, comBlidingFactor: bf}
}

func NewMembershipProver(witness *MembershipWitness, com, P *bn256.G1, Q *bn256.G2, PK []*bn256.G2, pp []*bn256.G1) *MembershipProver {
	return &MembershipProver{witness: witness, MembershipVerifier: NewMembershipVerifier(com, P, Q, PK, pp)}
}

func NewMembershipVerifier(com, P *bn256.G1, Q *bn256.G2, PK []*bn256.G2, pp []*bn256.G1) *MembershipVerifier {
	return &MembershipVerifier{PedersenParams: pp, CommitmentToValue: com, POKVerifier: &POKVerifier{PK: PK, Q: Q, P: P}}
}

// prover
type MembershipProver struct {
	*MembershipVerifier
	witness    *MembershipWitness
	randomness *MembershipRandomness
	Commitment *MembershipCommitment
}

// MembershipCommitment to randomness in proof
type MembershipCommitment struct {
	CommitmentToValue *bn256.G1
	Signature         *bn256.GT
}

// MembershipRandomness used in proof
type MembershipRandomness struct {
	value             *bn256.Zr
	comBlindingFactor *bn256.Zr
	sigBlindingFactor *bn256.Zr
	hash              *bn256.Zr
}

// verify whether a value has been signed
type MembershipVerifier struct {
	*POKVerifier
	PedersenParams    []*bn256.G1
	CommitmentToValue *bn256.G1
}

// generate a membership proof
func (p *MembershipProver) Prove() ([]byte, error) {
	if len(p.PK) != 3 {
		return nil, errors.Errorf("can't generate membership proof")
	}
	if len(p.PedersenParams) != 2 {
		return nil, errors.Errorf("can't generate membership proof")
	}
	proof := &MembershipProof{}
	proof.Commitment = p.CommitmentToValue
	// obfuscate signature
	var err error
	proof.Signature, err = p.obfuscateSignature()
	if err != nil {
		return nil, err
	}
	// compute hash
	p.computeHash()
	// compute commitment
	err = p.computeCommitment()
	if err != nil {
		return nil, err
	}
	// compute challenge
	proof.Challenge, err = p.computeChallenge(proof.Commitment, p.Commitment, proof.Signature)
	if err != nil {
		return nil, err
	}

	// generate proof
	sp := &common.SchnorrProver{Witness: []*bn256.Zr{p.witness.value, p.witness.comBlidingFactor, p.witness.hash, p.witness.sigBlindingFactor}, Randomness: []*bn256.Zr{p.randomness.value, p.randomness.comBlindingFactor, p.randomness.hash, p.randomness.sigBlindingFactor}, Challenge: proof.Challenge}
	proofs, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "range proof generation failed")
	}
	proof.Value = proofs[0]
	proof.ComBlindingFactor = proofs[1]
	proof.Hash = proofs[2]
	proof.SigBlindingFactor = proofs[3]

	return proof.Serialize()
}

// verify membership proof
func (v *MembershipVerifier) Verify(raw []byte) error {
	if len(v.PK) != 3 {
		return errors.Errorf("can't generate membership proof")
	}
	if len(v.PedersenParams) != 2 {
		return errors.Errorf("can't generate membership proof")
	}
	proof := &MembershipProof{}
	err := proof.Deserialize(raw)
	if err != nil {
		return err
	}

	com, err := v.recomputeCommitments(proof)
	if err != nil {
		return err
	}

	chal, err := v.computeChallenge(proof.Commitment, com, proof.Signature)
	if err != nil {
		return err
	}
	if chal.Cmp(proof.Challenge) != 0 {
		return errors.Errorf("invalid membership proof")
	}
	return nil
}

func (p *MembershipProver) obfuscateSignature() (*pssign.Signature, error) {
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	p.witness.sigBlindingFactor = bn256.RandModOrder(rand)
	err = p.witness.signature.Randomize()
	if err != nil {
		return nil, err
	}
	sig := &pssign.Signature{}
	sig.Copy(p.witness.signature)
	sig.S.Add(p.P.Mul(p.witness.sigBlindingFactor))

	return sig, nil
}

func (p *MembershipProver) computeCommitment() error {
	// Get RNG
	rand, err := bn256.GetRand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	// compute commitments
	p.randomness = &MembershipRandomness{}
	p.randomness.value = bn256.RandModOrder(rand)
	p.randomness.hash = bn256.RandModOrder(rand)
	p.randomness.sigBlindingFactor = bn256.RandModOrder(rand)

	t := p.PK[1].Mul(p.randomness.value)
	t.Add(p.PK[2].Mul(p.randomness.hash))

	p.Commitment = &MembershipCommitment{}
	p.Commitment.Signature = bn256.Pairing(t, p.witness.signature.R, p.Q, p.P.Mul(p.randomness.sigBlindingFactor))
	p.Commitment.Signature = bn256.FinalExp(p.Commitment.Signature)

	p.randomness.comBlindingFactor = bn256.RandModOrder(rand)
	p.Commitment.CommitmentToValue = p.PedersenParams[0].Mul(p.randomness.value)
	p.Commitment.CommitmentToValue.Add(p.PedersenParams[1].Mul(p.randomness.comBlindingFactor))

	return nil
}

func (v *MembershipVerifier) computeChallenge(comToValue *bn256.G1, com *MembershipCommitment, signature *pssign.Signature) (*bn256.Zr, error) {
	g1array := common.GetG1Array(v.PedersenParams, []*bn256.G1{comToValue, com.CommitmentToValue, v.P})
	g2array := common.GetG2Array(v.PK, []*bn256.G2{v.Q})
	raw := common.GetBytesArray(g1array.Bytes(), g2array.Bytes(), com.Signature.Bytes())
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to compute challenge: error while serializing Pointcheval-Sanders signature")
	}
	raw = append(raw, bytes...)

	return bn256.HashModOrder(raw), nil
}

func (p *MembershipProver) computeHash() {
	bytes := p.witness.value.Bytes()
	p.witness.hash = bn256.HashModOrder(bytes)
	return
}

// recompute commitments for verification
func (v *MembershipVerifier) recomputeCommitments(p *MembershipProof) (*MembershipCommitment, error) {
	psv := &POKVerifier{P: v.P, Q: v.Q, PK: v.PK}
	c := &MembershipCommitment{}

	psp := &POK{
		Challenge:      p.Challenge,
		Signature:      p.Signature,
		Messages:       []*bn256.Zr{p.Value},
		Hash:           p.Hash,
		BlindingFactor: p.SigBlindingFactor,
	}
	var err error
	c.Signature, err = psv.RecomputeCommitment(psp)
	if err != nil {
		return nil, err
	}
	ver := &common.SchnorrVerifier{PedParams: v.PedersenParams}
	zkp := &common.SchnorrProof{Statement: v.CommitmentToValue, Proof: []*bn256.Zr{p.Value, p.ComBlindingFactor}, Challenge: p.Challenge}
	c.CommitmentToValue = ver.RecomputeCommitment(zkp)

	return c, nil
}
