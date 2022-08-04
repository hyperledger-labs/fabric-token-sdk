/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

// MembershipProof is a ZK proof that shows that a committed value is signed using
// Pointcheval-Sanders signature
type MembershipProof struct {
	// Challenge is the challenge computed for the ZK proof
	Challenge *math.Zr
	// Obfuscated Pointcheval-Sanders Signature
	Signature *pssign.Signature
	// Proof of knowledge of committed value
	Value *math.Zr
	// Proof of knowledge of the blinding factor in the Pedersen commitment
	ComBlindingFactor *math.Zr
	// Proof of knowledge of the blinding factor used to obfuscate Pointcheval-Sanders signature
	SigBlindingFactor *math.Zr
	// Proof of knowledge of the hash signed in Pointcheval-Sanders signature
	Hash *math.Zr
	// Pedersen commitment to Value
	Commitment *math.G1
}

// Serialize marshals MembershipProof
func (p *MembershipProof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize un-marshals MembershipProof
func (p *MembershipProof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

// MembershipWitness contains the information needed to generate a MembershipProof
type MembershipWitness struct {
	// Pointchval-Sanders signature on value
	signature *pssign.Signature
	// value is the signed message
	value *math.Zr
	// hash is hash of value and it is also signed with Pointcheval-Sandes signature
	hash *math.Zr
	//lint:ignore U1000 TODO @Kaoutar: Is this field still needed?
	// sigBlindingFactor is the randomness used to obfuscate signature
	sigBlindingFactor *math.Zr
	// comBlindingFactor is the randomness used to compute the Pedersen commitment of value
	comBlindingFactor *math.Zr
}

// NewMembershipWitness returns a MembershipWitness as a function of the passed arguments
func NewMembershipWitness(sig *pssign.Signature, value *math.Zr, bf *math.Zr) *MembershipWitness {
	return &MembershipWitness{signature: sig, value: value, comBlindingFactor: bf}
}

// NewMembershipProver returns a MembershipWitnessProver for the passed MembershipWitness
func NewMembershipProver(witness *MembershipWitness, com, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *MembershipProver {
	return &MembershipProver{witness: witness, MembershipVerifier: NewMembershipVerifier(com, P, Q, PK, pp, curve)}
}

// NewMembershipVerifier returns a MembershipVerifier for the passed commitment com
// The verifier checks if the committed value in com is signed using Pointcheval-Sanders
// and the signature verifies correctly relative to the passed public key PK
func NewMembershipVerifier(com, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *MembershipVerifier {
	return &MembershipVerifier{PedersenParams: pp, CommitmentToValue: com, POKVerifier: &POKVerifier{PK: PK, Q: Q, P: P, Curve: curve}}
}

// MembershipProver is a ZK prover that shows that a committed value is signed with
// Pointcheval-Sanders signature
type MembershipProver struct {
	*MembershipVerifier
	witness *MembershipWitness
}

// MembershipCommitment is commitment to randomness used to compute MembershipProof
type MembershipCommitment struct {
	CommitmentToValue *math.G1
	Signature         *math.Gt
}

// MembershipRandomness is randomness used to compute MembershipProof
type MembershipRandomness struct {
	// randomness used to compute proof of value
	value *math.Zr
	// randomness used to compute proof of comBlindingFactor
	comBlindingFactor *math.Zr
	// randomness used to compute proof of sigBlindingFactor
	sigBlindingFactor *math.Zr
	// randomness used to compute proof of hash
	hash *math.Zr
}

// MembershipVerifier verifies whether a committed value has been signed using
// Pointcheval-Sanders signature
type MembershipVerifier struct {
	*POKVerifier
	PedersenParams    []*math.G1
	CommitmentToValue *math.G1
}

// Prove produces a MembershipProof
func (p *MembershipProver) Prove() (*MembershipProof, error) {
	proof := &MembershipProof{}
	proof.Commitment = p.CommitmentToValue

	var err error
	// obfuscate Pointcheval-Sanders signature
	obfuscatedSignature, err := p.obfuscateSignature()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate membership proof")
	}
	// compute hash of value
	if p.witness.value == nil {
		return nil, errors.New("failed to generate membership proof: nil value")
	}
	p.witness.hash = p.Curve.HashToZr(p.witness.value.Bytes())

	// compute randomness and commitment to randomness
	commitment, randomness, err := p.computeCommitment(obfuscatedSignature.randomizedWitnessSignature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate membership proof")
	}

	// compute challenge
	proof.Challenge, err = p.computeChallenge(proof.Commitment, commitment, obfuscatedSignature.obfuscatedSig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate membership proof")
	}

	if p.witness == nil {
		return nil, errors.New("please initialize witness of membership proof")
	}
	// generate ZK proof
	sp := &common.SchnorrProver{Witness: []*math.Zr{p.witness.value, p.witness.comBlindingFactor, p.witness.hash, obfuscatedSignature.blindingFactor}, Randomness: []*math.Zr{randomness.value, randomness.comBlindingFactor, randomness.hash, randomness.sigBlindingFactor}, Challenge: proof.Challenge, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	proofs, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate membership proof")
	}

	// instantiate proof
	proof.Signature = obfuscatedSignature.obfuscatedSig
	proof.Value = proofs[0]
	proof.ComBlindingFactor = proofs[1]
	proof.Hash = proofs[2]
	proof.SigBlindingFactor = proofs[3]

	return proof, nil
}

// Verify checks the validity of a serialized MembershipProof
// Verify returns an error if the serialized MembershipProof is invalid
func (v *MembershipVerifier) Verify(proof *MembershipProof) error {
	// recompute commitments to randomness used in MembershipProof
	com, err := v.recomputeCommitments(proof)
	if err != nil {
		return errors.Wrap(err, "failed to verify membership proof")
	}

	// compute challenge
	chal, err := v.computeChallenge(proof.Commitment, com, proof.Signature)
	if err != nil {
		return errors.Wrap(err, "failed to verify membership proof")
	}

	// check if MembershipProof is valid
	if !chal.Equals(proof.Challenge) {
		return errors.New("invalid membership proof")
	}
	return nil
}

// obfuscatedSignature contains an obfuscated Pointcheval-Sanders signature
type obfuscatedSignature struct {
	// randomness used to obfuscate signature
	blindingFactor *math.Zr
	// this is a randomized Pointcheval-Sanders signature
	// sigma' =(R', S') = (R^r, S^r) = sigma^r
	randomizedWitnessSignature *pssign.Signature
	// obfuscated signature
	// sigma'' = (R', S'*G^r)
	obfuscatedSig *pssign.Signature
}

// obfuscateSignature return an obfuscatedSignature to be used used in generating
// a MembershipProof
func (p *MembershipProver) obfuscateSignature() (*obfuscatedSignature, error) {
	if p.Curve == nil {
		return nil, errors.New("cannot obfuscate signature: please initialize curve")
	}
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.New("failed to get random number generator to obfuscate signature")
	}

	blindingFactor := p.Curve.NewRandomZr(rand)
	v := pssign.NewVerifier(nil, nil, p.Curve)
	randomizedWitnessSignature := &pssign.Signature{}
	randomizedWitnessSignature.Copy(p.witness.signature)
	err = v.Randomize(randomizedWitnessSignature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to randomize signature")
	}
	sig := &pssign.Signature{}
	sig.Copy(randomizedWitnessSignature)
	sig.S.Add(p.P.Mul(blindingFactor))

	return &obfuscatedSignature{
		randomizedWitnessSignature: randomizedWitnessSignature,
		blindingFactor:             blindingFactor,
		obfuscatedSig:              sig,
	}, nil
}

// computeCommitment returns MembershipCommitment and MembershipRandomness
func (p *MembershipProver) computeCommitment(obfuscatedSignature *pssign.Signature) (*MembershipCommitment, *MembershipRandomness, error) {
	// Get RNG
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, errors.Errorf("failed to get RNG")
	}
	// compute commitments
	randomness := &MembershipRandomness{}
	randomness.value = p.Curve.NewRandomZr(rand)
	randomness.hash = p.Curve.NewRandomZr(rand)
	randomness.sigBlindingFactor = p.Curve.NewRandomZr(rand)
	if len(p.PK) != 3 {
		return nil, nil, errors.New("failed to compute commitment: invalid public key")
	}
	t := p.PK[1].Mul(randomness.value)
	t.Add(p.PK[2].Mul(randomness.hash))

	if p.Q == nil || p.P == nil {
		return nil, nil, errors.New("failed to compute commitment: invalid public parameters")
	}
	commitment := &MembershipCommitment{}
	commitment.Signature = p.Curve.Pairing2(t, obfuscatedSignature.R, p.Q, p.P.Mul(randomness.sigBlindingFactor))
	commitment.Signature = p.Curve.FExp(commitment.Signature)

	if len(p.PedersenParams) != 2 {
		return nil, nil, errors.New("failed to compute commitment: invalid Pedersen parameters")
	}
	randomness.comBlindingFactor = p.Curve.NewRandomZr(rand)
	commitment.CommitmentToValue = p.PedersenParams[0].Mul(randomness.value)
	commitment.CommitmentToValue.Add(p.PedersenParams[1].Mul(randomness.comBlindingFactor))

	return commitment, randomness, nil
}

// computeChallenge computes the challenge for MembershipProof
func (v *MembershipVerifier) computeChallenge(comToValue *math.G1, com *MembershipCommitment, signature *pssign.Signature) (*math.Zr, error) {
	g1array, err := common.GetG1Array(v.PedersenParams, []*math.G1{comToValue, com.CommitmentToValue, v.P}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	g2array, err := common.GetG2Array(v.PK, []*math.G2{v.Q}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	raw := common.GetBytesArray(g1array, g2array, com.Signature.Bytes())
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to compute challenge: error while serializing Pointcheval-Sanders signature")
	}
	raw = append(raw, bytes...)

	return v.Curve.HashToZr(raw), nil
}

// recomputeCommitments recompute commitments to the randomness used in the MembershipProof
// result is used to verify MembershipProof
func (v *MembershipVerifier) recomputeCommitments(p *MembershipProof) (*MembershipCommitment, error) {
	psv := &POKVerifier{P: v.P, Q: v.Q, PK: v.PK, Curve: v.Curve}
	c := &MembershipCommitment{}

	psp := &POK{
		Challenge:      p.Challenge,
		Signature:      p.Signature,
		Messages:       []*math.Zr{p.Value},
		Hash:           p.Hash,
		BlindingFactor: p.SigBlindingFactor,
	}
	var err error
	c.Signature, err = psv.recomputeCommitment(psp)
	if err != nil {
		return nil, err
	}
	ver := &common.SchnorrVerifier{PedParams: v.PedersenParams, Curve: v.Curve}
	zkp := &common.SchnorrProof{Statement: v.CommitmentToValue, Proof: []*math.Zr{p.Value, p.ComBlindingFactor}, Challenge: p.Challenge}
	c.CommitmentToValue, err = ver.RecomputeCommitment(zkp)
	if err != nil {
		return nil, err
	}

	return c, nil
}
