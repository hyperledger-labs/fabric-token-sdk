/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

type SigProof struct {
	Challenge         *math.Zr
	Hidden            []*math.Zr
	Hash              *math.Zr
	Signature         *pssign.Signature
	SigBlindingFactor *math.Zr
	ComBlindingFactor *math.Zr
	Commitment        *math.G1 // for hidden values
}

// commitments how they are computed
type SigProver struct {
	*SigVerifier
	witness    *SigWitness
	randomness *SigRandomness
	Commitment *SigCommitment
}

type SigVerifier struct {
	*POKVerifier
	HiddenIndices        []int
	DisclosedIndices     []int
	Disclosed            []*math.Zr
	PedersenParams       []*math.G1
	CommitmentToMessages *math.G1
}

type SigWitness struct {
	hidden            []*math.Zr
	hash              *math.Zr
	signature         *pssign.Signature
	sigBlindingFactor *math.Zr
	comBlindingFactor *math.Zr
}

func NewSigWitness(hidden []*math.Zr, signature *pssign.Signature, hash, bf *math.Zr) *SigWitness {
	return &SigWitness{
		hidden:            hidden,
		hash:              hash,
		signature:         signature,
		comBlindingFactor: bf,
	}
}

func NewSigProver(hidden, disclosed []*math.Zr, signature *pssign.Signature, hash, bf *math.Zr, com *math.G1, hiddenindices, disclosedindices []int, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *SigProver {
	return &SigProver{
		witness:     NewSigWitness(hidden, signature, hash, bf),
		SigVerifier: NewSigVerifier(hiddenindices, disclosedindices, disclosed, com, P, Q, PK, pp, curve),
	}
}

func NewSigVerifier(hidden, disclosed []int, disclosedInf []*math.Zr, com, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *SigVerifier {
	return &SigVerifier{
		POKVerifier: &POKVerifier{
			P:     P,
			Q:     Q,
			PK:    PK,
			Curve: curve,
		},
		HiddenIndices:        hidden,
		DisclosedIndices:     disclosed,
		Disclosed:            disclosedInf,
		CommitmentToMessages: com,
		PedersenParams:       pp,
	}
}

type SigRandomness struct {
	hidden            []*math.Zr
	hash              *math.Zr
	sigBlindingFactor *math.Zr
	comBlindingFactor *math.Zr
}

type SigCommitment struct {
	CommitmentToMessages *math.G1
	Signature            *math.Gt
}

func (p *SigProver) Prove() (*SigProof, error) {
	if len(p.HiddenIndices) != len(p.witness.hidden) {
		return nil, errors.Errorf("witness is not of the right size")
	}
	if len(p.DisclosedIndices) != len(p.Disclosed) {
		return nil, errors.Errorf("witness is not of the right size")
	}
	proof := &SigProof{}
	var err error
	// randomize signature
	proof.Signature, err = p.obfuscateSignature()
	if err != nil {
		return nil, err
	}
	err = p.computeCommitment()
	if err != nil {
		return nil, err
	}

	// compute challenge
	proof.Challenge, err = p.computeChallenge(p.CommitmentToMessages, proof.Signature, p.Commitment)
	if err != nil {
		return nil, err
	}
	// compute proofs
	sp := &common.SchnorrProver{Witness: append(p.witness.hidden, p.witness.comBlindingFactor, p.witness.sigBlindingFactor, p.witness.hash), Randomness: append(p.randomness.hidden, p.randomness.comBlindingFactor, p.randomness.sigBlindingFactor, p.randomness.hash), Challenge: proof.Challenge, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	proofs, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "signature proof generation failed")
	}

	proof.Commitment = p.CommitmentToMessages
	proof.Hidden = proofs[:len(p.witness.hidden)]
	proof.ComBlindingFactor = proofs[len(p.witness.hidden)]
	proof.SigBlindingFactor = proofs[len(p.witness.hidden)+1]
	proof.Hash = proofs[len(p.witness.hidden)+2]

	return proof, nil
}

func (p *SigProver) computeCommitment() error {
	if len(p.PedersenParams) != len(p.HiddenIndices)+1 {
		return errors.Errorf("size of witness does not match length of Pedersen Parameters")
	}
	if len(p.PK) != len(p.HiddenIndices)+len(p.DisclosedIndices)+2 {
		return errors.Errorf("size of signature public key does not mathc the size of the witness")
	}
	// Get RNG
	rand, err := p.Curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	// generate randomness
	p.randomness = &SigRandomness{}
	for i := 0; i < len(p.witness.hidden); i++ {
		p.randomness.hidden = append(p.randomness.hidden, p.Curve.NewRandomZr(rand))
	}
	p.randomness.hash = p.Curve.NewRandomZr(rand)
	p.randomness.comBlindingFactor = p.Curve.NewRandomZr(rand)
	p.randomness.sigBlindingFactor = p.Curve.NewRandomZr(rand)

	// compute commitment
	p.Commitment = &SigCommitment{}
	p.Commitment.CommitmentToMessages = p.PedersenParams[len(p.witness.hidden)].Mul(p.randomness.comBlindingFactor)
	for i, r := range p.randomness.hidden {
		p.Commitment.CommitmentToMessages.Add(p.PedersenParams[i].Mul(r))
	}

	t := p.PK[len(p.Disclosed)+len(p.witness.hidden)+1].Mul(p.randomness.hash)
	for i, index := range p.HiddenIndices {
		t.Add(p.PK[index+1].Mul(p.randomness.hidden[i]))
	}

	p.Commitment.Signature = p.Curve.Pairing2(t, p.witness.signature.R, p.Q, p.P.Mul(p.randomness.sigBlindingFactor))
	p.Commitment.Signature = p.Curve.FExp(p.Commitment.Signature)
	return nil
}

func (p *SigProver) obfuscateSignature() (*pssign.Signature, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	p.witness.sigBlindingFactor = p.Curve.NewRandomZr(rand)
	v := &pssign.SignVerifier{Curve: p.Curve}
	err = v.Randomize(p.witness.signature)
	if err != nil {
		return nil, err
	}
	sig := &pssign.Signature{}
	sig.Copy(p.witness.signature)
	sig.S.Add(p.P.Mul(p.witness.sigBlindingFactor))

	return sig, nil
}

func (v *SigVerifier) computeChallenge(comToMessages *math.G1, signature *pssign.Signature, com *SigCommitment) (*math.Zr, error) {
	g1array := common.GetG1Array(v.PedersenParams, []*math.G1{comToMessages, com.CommitmentToMessages,
		v.P})
	g2array := common.GetG2Array(v.PK, []*math.G2{v.Q})
	raw := common.GetBytesArray(g1array.Bytes(), g2array.Bytes(), com.Signature.Bytes())
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to compute challenge: error while serializing Pointcheval-Sanders signature")
	}
	raw = append(raw, bytes...)

	return v.Curve.HashToZr(raw), nil
}

// recompute commitments for verification
func (v *SigVerifier) recomputeCommitments(p *SigProof) (*SigCommitment, error) {
	if len(p.Hidden)+len(v.Disclosed) != len(v.PK)-2 {
		return nil, errors.Errorf("length of signature public key does not match number of signed messages")
	}

	c := &SigCommitment{}
	ver := &common.SchnorrVerifier{PedParams: v.PedersenParams, Curve: v.Curve}
	zkp := &common.SchnorrProof{Statement: v.CommitmentToMessages, Proof: append(p.Hidden, p.ComBlindingFactor), Challenge: p.Challenge}

	c.CommitmentToMessages = ver.RecomputeCommitment(zkp)
	proof := make([]*math.Zr, len(v.PK)-2)
	for i, index := range v.HiddenIndices {
		proof[index] = p.Hidden[i]
	}
	for i, index := range v.DisclosedIndices {
		proof[index] = v.Curve.ModMul(v.Disclosed[i], p.Challenge, v.Curve.GroupOrder)
	}

	sp := &POK{
		Challenge:      p.Challenge,
		Signature:      p.Signature,
		Messages:       proof,
		Hash:           p.Hash,
		BlindingFactor: p.SigBlindingFactor,
	}

	sv := &POKVerifier{P: v.P, Q: v.Q, PK: v.PK, Curve: v.Curve}
	var err error
	c.Signature, err = sv.RecomputeCommitment(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify signature proof")
	}
	return c, nil
}

// verify membership proof
func (v *SigVerifier) Verify(p *SigProof) error {

	com, err := v.recomputeCommitments(p)
	if err != nil {
		return nil
	}

	chal, err := v.computeChallenge(p.Commitment, p.Signature, com)
	if err != nil {
		return nil
	}
	if !chal.Equals(p.Challenge) {
		return errors.Errorf("invalid signature proof")
	}
	return nil
}
