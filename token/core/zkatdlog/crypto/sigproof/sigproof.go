/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

// SigProof is a proof of knowledge of Pointcheval-Signature
// with partial message disclosure
type SigProof struct {
	// Challenge is the challenge used in the proof
	Challenge *math.Zr
	// Hidden is an array that contains proofs of knowledge of hidden messages
	Hidden []*math.Zr
	// Hash is the proof of knowledge of the hash signed in Pointcheval-Sanders signature
	// If hash is computed as a function of the signed messages, it should never be disclosed
	// Hash cannot be nil
	Hash *math.Zr
	// Signature is an obfuscated Pointcheval-Sanders signature
	Signature *pssign.Signature
	// SigBlindingFactor is the proof of knowledge of the randomness used to obfuscate
	// Pointcheval-Sanders signature
	SigBlindingFactor *math.Zr
	// ComBlindingFactor is a proof of knowledge of the randomness used to compute
	// the Pedersen commitment
	ComBlindingFactor *math.Zr
	// Commitment is a Pedersen commitment of to the hidden messages
	Commitment *math.G1 // for hidden values
}

// SigProver produces a proof of knowledge Pointcheval-Sanders signature with partial
// message disclosure
type SigProver struct {
	*SigVerifier
	witness    *SigWitness
	randomness *SigRandomness
	Commitment *SigCommitment
}

// SigVerifier checks the validity of SigProof
type SigVerifier struct {
	*POKVerifier
	// indices of messages to be hidden in the signed vector
	HiddenIndices []int
	// indices of messages to be disclosed
	DisclosedIndices []int
	// Disclosed contains the content of the disclosed messages
	Disclosed []*math.Zr
	// PedersenParams corresponds to Pedersen commitment generators
	PedersenParams []*math.G1
	// CommitmentToMessages is a Pedersen commitment to the signed messages
	CommitmentToMessages *math.G1
}

// SigWitness is the witness for SigProof
type SigWitness struct {
	hidden            []*math.Zr
	hash              *math.Zr
	signature         *pssign.Signature
	sigBlindingFactor *math.Zr
	comBlindingFactor *math.Zr
}

// NewSigWitness instantiates a SigWitness with the passed arguments
func NewSigWitness(hidden []*math.Zr, signature *pssign.Signature, hash, bf *math.Zr) *SigWitness {
	return &SigWitness{
		hidden:            hidden,
		hash:              hash,
		signature:         signature,
		comBlindingFactor: bf,
	}
}

// NewSigProver returns a SigProver as a function of the passed arguments
func NewSigProver(hidden, disclosed []*math.Zr, signature *pssign.Signature, hash, bf *math.Zr, com *math.G1, hiddenindices, disclosedindices []int, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *SigProver {
	return &SigProver{
		witness:     NewSigWitness(hidden, signature, hash, bf),
		SigVerifier: NewSigVerifier(hiddenindices, disclosedindices, disclosed, com, P, Q, PK, pp, curve),
	}
}

// NewSigVerifier returns a SigVerifier as a function of the passed arguments
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

// SigRandomness represents the randomness used in the SigProof
type SigRandomness struct {
	hidden            []*math.Zr
	hash              *math.Zr
	sigBlindingFactor *math.Zr
	comBlindingFactor *math.Zr
}

// SigCommitment encodes the commitments to the randomness used in the SigProof
type SigCommitment struct {
	CommitmentToMessages *math.G1
	Signature            *math.Gt
}

// Prove returns a SigProof
func (p *SigProver) Prove() (*SigProof, error) {

	proof := &SigProof{}
	var err error
	// randomize signature
	proof.Signature, err = p.obfuscateSignature()
	if err != nil {
		return nil, err
	}

	// generate randomness and compute corresponding commitments
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
		return nil, errors.Wrap(err, "signature proof generation failed")
	}

	proof.Commitment = p.CommitmentToMessages
	proof.Hidden = proofs[:len(p.witness.hidden)]
	proof.ComBlindingFactor = proofs[len(p.witness.hidden)]
	proof.SigBlindingFactor = proofs[len(p.witness.hidden)+1]
	proof.Hash = proofs[len(p.witness.hidden)+2]

	return proof, nil
}

// computeCommitment computes the commitments to the randomness used in the SigProof
func (p *SigProver) computeCommitment() error {

	// Get random number generator
	rand, err := p.Curve.Rand()
	if err != nil {
		return errors.New("failed to get random number generator")
	}

	// generate randomness
	p.randomness = &SigRandomness{}
	for i := 0; i < len(p.witness.hidden); i++ {
		p.randomness.hidden = append(p.randomness.hidden, p.Curve.NewRandomZr(rand))
	}
	p.randomness.hash = p.Curve.NewRandomZr(rand)
	p.randomness.comBlindingFactor = p.Curve.NewRandomZr(rand)
	p.randomness.sigBlindingFactor = p.Curve.NewRandomZr(rand)

	if len(p.PedersenParams) != len(p.HiddenIndices)+1 {
		return errors.New("size of witness does not match length of Pedersen Parameters")
	}

	// compute commitments to randomness
	p.Commitment = &SigCommitment{}
	if p.PedersenParams[len(p.witness.hidden)] == nil {
		return errors.New("please initialize Pedersen parameters")
	}
	p.Commitment.CommitmentToMessages = p.PedersenParams[len(p.witness.hidden)].Mul(p.randomness.comBlindingFactor)
	for i, r := range p.randomness.hidden {
		if p.PedersenParams[i] == nil {
			return errors.New("please initialize Pedersen parameters")
		}
		p.Commitment.CommitmentToMessages.Add(p.PedersenParams[i].Mul(r))
	}

	if len(p.PK) != len(p.HiddenIndices)+len(p.DisclosedIndices)+2 {
		return errors.Errorf("size of signature public key does not mathc the size of the witness")
	}

	if p.PK[len(p.Disclosed)+len(p.witness.hidden)+1] == nil {
		return errors.New("please initialize public keys")
	}
	t := p.PK[len(p.Disclosed)+len(p.witness.hidden)+1].Mul(p.randomness.hash)
	for i, index := range p.HiddenIndices {
		if p.PK[index+1] == nil {
			return errors.New("please initialize public keys")
		}
		t.Add(p.PK[index+1].Mul(p.randomness.hidden[i]))
	}

	p.Commitment.Signature = p.Curve.Pairing2(t, p.witness.signature.R, p.Q, p.P.Mul(p.randomness.sigBlindingFactor))
	p.Commitment.Signature = p.Curve.FExp(p.Commitment.Signature)
	return nil
}

// obfuscateSignature obfuscates a Pointcheval-Sanders signature
func (p *SigProver) obfuscateSignature() (*pssign.Signature, error) {
	if p.Curve == nil {
		return nil, errors.New("please initialize curve")
	}
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.New("failed to get random number generator")
	}
	if p.witness == nil {
		return nil, errors.New("please initialize witness")
	}

	p.witness.sigBlindingFactor = p.Curve.NewRandomZr(rand)
	v := &pssign.SignVerifier{Curve: p.Curve}
	err = v.Randomize(p.witness.signature)
	if err != nil {
		return nil, err
	}
	if p.P == nil {
		return nil, errors.New("please initialize public parameters")
	}
	sig := &pssign.Signature{}
	sig.Copy(p.witness.signature)
	sig.S.Add(p.P.Mul(p.witness.sigBlindingFactor))

	return sig, nil
}

// computeChallenge returns the challenge for the SigProof
func (v *SigVerifier) computeChallenge(comToMessages *math.G1, signature *pssign.Signature, com *SigCommitment) (*math.Zr, error) {
	g1array, err := common.GetG1Array(v.PedersenParams, []*math.G1{comToMessages, com.CommitmentToMessages,
		v.P}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	g2array, err := common.GetG2Array(v.PK, []*math.G2{v.Q}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	if com.Signature == nil {
		return nil, errors.New("failed to compute challenge:commitment is not well formed")
	}
	raw := common.GetBytesArray(g1array, g2array, com.Signature.Bytes())
	if signature == nil {
		return nil, errors.New("failed to compute challenge: Pointcheval-Sanders signature is nil")
	}
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to compute challenge: error while serializing Pointcheval-Sanders signature")
	}
	raw = append(raw, bytes...)

	return v.Curve.HashToZr(raw), nil
}

// recomputeCommitments returns the commitments to randomness in the SigProof
func (v *SigVerifier) recomputeCommitments(p *SigProof) (*SigCommitment, error) {

	/**/
	c := &SigCommitment{}
	ver := &common.SchnorrVerifier{PedParams: v.PedersenParams, Curve: v.Curve}
	zkp := &common.SchnorrProof{Statement: v.CommitmentToMessages, Proof: append(p.Hidden, p.ComBlindingFactor), Challenge: p.Challenge}
	var err error
	c.CommitmentToMessages, err = ver.RecomputeCommitment(zkp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to recompute commitment")
	}
	if len(p.Hidden)+len(v.Disclosed) != len(v.PK)-2 {
		return nil, errors.New("invalid signature proof")
	}
	proof := make([]*math.Zr, len(v.PK)-2)
	for i, index := range v.HiddenIndices {
		proof[index] = p.Hidden[i]
	}
	for i, index := range v.DisclosedIndices {
		if v.Disclosed[i] == nil || p.Challenge == nil {
			return nil, errors.New("signature proof is not well formed")
		}
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

	c.Signature, err = sv.recomputeCommitment(sp)
	if err != nil {
		return nil, errors.Wrap(err, "invalid signature proof")
	}
	return c, nil
}

// Verify returns an error if the passed SigProof is invalid
func (v *SigVerifier) Verify(p *SigProof) error {
	if p == nil {
		return errors.New("invalid signature proof")
	}
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
