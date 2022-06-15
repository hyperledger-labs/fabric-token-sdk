/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/elgamal"
	"github.com/pkg/errors"
)

// Recipient requests a Pointcheval-Sanders blind signature
// Recipient would like to obtain a Pointcheval-Sanders signature on
// a committed vector of messages
type Recipient struct {
	*encVerifier
	*SignVerifier
	// Elgamal encryption secret key
	// This is used to decrypt the blind signature
	EncSK *elgamal.SecretKey
	// encWitness coresponds to the message and the randomness used to
	// encrypt the blind signature request
	Witness *encWitness
	// Elliptic curve
	Curve *math.Curve
}

// BlindSigner produces Pointcheval-Sanders blind signatures
type BlindSigner struct {
	// Signer is a Pointcheval-Sanders signer
	*Signer
	// PedersenParameters is the generators used to commit to the vector
	// of messages to be signed
	PedersenParameters []*math.G1
}

// NewBlindSigner returns a Pointcheval-Sanders BlindSigner
// as a function of the passed arguments
func NewBlindSigner(SK []*math.Zr, PK []*math.G2, Q *math.G2, pp []*math.G1, curve *math.Curve) *BlindSigner {
	s := &BlindSigner{PedersenParameters: pp}
	s.Signer = NewSigner(SK, PK, Q, curve)
	return s
}

// NewRecipient returns a Recipient that would like to obtain
// a Pointcheval-Sanders blind signature on the passed messages
func NewRecipient(messages []*math.Zr, blindingfactor *math.Zr, com *math.G1, sk *math.Zr, gen, pk *math.G1, pp []*math.G1, PK []*math.G2, Q *math.G2, curve *math.Curve) *Recipient {
	return &Recipient{
		Witness: &encWitness{
			messages:          messages,
			comBlindingFactor: blindingfactor,
		},
		EncSK: elgamal.NewSecretKey(sk, gen, pk, curve),
		encVerifier: &encVerifier{
			PedersenParameters: pp,
			Commitment:         com,
			Curve:              curve,
		},
		SignVerifier: &SignVerifier{
			PK:    PK,
			Q:     Q,
			Curve: curve,
		},
		Curve: curve,
	}
}

// encVerifier verifies if a vector of Elgamal ciphertexts are the encryption
// of a committed vector
type encVerifier struct {
	// Pedersen commitment generators
	PedersenParameters []*math.G1 //g_0, g_1, g_2, g_3, h (owner, type, value, sn, randomness)
	// commitment to encrypted messages
	Commitment *math.G1
	// encryption of messages
	Ciphertexts []*elgamal.Ciphertext
	// One-time Elgamal public key
	EncPK *elgamal.PublicKey
	// Elliptic curve
	Curve *math.Curve
}

// encWitness if the secret information that the recipient uses to produce EncProof
type encWitness struct {
	// committed messages and which will be encrypted
	messages []*math.Zr
	// randomness used in encryption of messages
	encRandomness []*math.Zr
	// randomness used to compute the commitment to messages
	comBlindingFactor *math.Zr
}

// EncProof is a zero-knowledge proof of correct encryption of committed messages
// It consists of zero-knowledge proofs of knowledge of messages that open a commitment
// and proofs of correct encryption of the same messages under a known Elgamal public key
type EncProof struct {
	// ZKP of knowledge of committed/encrypted messages
	Messages []*math.Zr
	// ZKP of knowledge of randomness used in the encryption
	EncRandomness []*math.Zr
	// ZKP of knowledge of the randomness (blinding factor) used in the commitment
	ComBlindingFactor *math.Zr
	// ZKP challenge
	Challenge *math.Zr
}

// BlindSignRequest is what the recipient send to the Pointcheval-Sanders signer to
// obtain the blind signature
type BlindSignRequest struct {
	// Pedersen commitment of a vector of messages
	Commitment *math.G1
	// Elgamal encryption of the committed messages
	Ciphertexts []*elgamal.Ciphertext
	// Proof of correctness of encryption and commitment
	Proof *EncProof
	// One-time Elgamal public key picked by the recipient
	EncPK *elgamal.PublicKey
}

// BlindSignResponse is the response of the BlindSigner to a blind signature request
type BlindSignResponse struct {
	// hash used in the  Pointcheval-Sanders signature
	Hash *math.Zr
	// this encrypts the Pointcheval-Sanders signature
	Ciphertext *elgamal.Ciphertext
}

// encProofRandomness corresponds to the randomness used in the generation of EncProof
type encProofRandomness struct {
	// randomness used to compute the ZKP of knowledge of messages
	messages []*math.Zr
	// randomness used to compute the ZKP of knowledge of encryption randomness
	encRandomness []*math.Zr
	// randomness used to compute the ZKP of knowledge of commitment randomness
	comBlindingFactor *math.Zr
}

// EncProofCommitments contains the commitments to EncProof randomness
// For a statement (x1, ..., x_n): y = \prod_{i=1}^n g_i^x_i, one computes
// s = \prod_{i=1}^n g_i^r_i as the commitment to randomness (r_1, ..., r_n)
type EncProofCommitments struct {
	C1         []*math.G1
	C2         []*math.G1
	Commitment *math.G1
}

// BlindSign takes as input a BlindSignRequest and returns the corresponding BlindSignResponse,
// if the request is valid. Else, BlindSign returns an error
func (s *BlindSigner) BlindSign(request *BlindSignRequest) (*BlindSignResponse, error) {
	if request == nil {
		return nil, errors.New("cannot produce Pointcheval-Sanders signature: nil blind signature request")
	}
	if s.Curve == nil {
		return nil, errors.New("cannot produce Pointcheval-Sanders signature: please initialize curve")
	}
	if len(request.Ciphertexts) != len(s.PK)-2 {
		return nil, errors.Errorf("cannot produce Pointcheval-Sanders signature: number of ciphertexts request does not match number of public keys: expect [%d], got [%d]", len(s.PK)-2, len(request.Ciphertexts))
	}
	// verify encryption correctness
	v := &encVerifier{Commitment: request.Commitment, Ciphertexts: request.Ciphertexts, EncPK: request.EncPK, PedersenParameters: s.PedersenParameters, Curve: s.Curve}

	err := v.Verify(request.Proof)
	if err != nil {
		return nil, errors.New("cannot produce Pointcheval-Sanders signature: invalid request")
	}

	raw, err := json.Marshal(request.Proof)
	if err != nil {
		return nil, err
	}
	// hash proof in request
	// this will be blindly signed along the messages
	// this is an artefact of Pointcheval-Sanders signature
	response := &BlindSignResponse{Hash: s.Curve.HashToZr(raw)}
	// this results in an Elgamal encryption of a Pointcheval-Sanders signature
	response.Ciphertext = &elgamal.Ciphertext{}
	// generator for Pointcheval-Sanders signature
	hash := s.Signer.Curve.HashToG1(request.Commitment.Bytes())
	response.Ciphertext.C1 = s.Curve.NewG1()
	if s.SK[0] == nil {
		return nil, errors.New("cannot produce Pointcheval-Sanders signature: please initialize signer secret keys")
	}
	response.Ciphertext.C2 = hash.Mul(s.SK[0])
	for i := 0; i < len(request.Ciphertexts); i++ {
		if s.SK[i+1] == nil {
			return nil, errors.New("cannot produce Pointcheval-Sanders signature: please initialize signer secret keys")
		}
		response.Ciphertext.C1.Add(request.Ciphertexts[i].C1.Mul(s.SK[i+1]))
		response.Ciphertext.C2.Add(request.Ciphertexts[i].C2.Mul(s.SK[i+1]))
	}
	if s.SK[len(request.Ciphertexts)+1] == nil {
		return nil, errors.New("cannot produce Pointcheval-Sanders signature: please initialize signer secret keys")
	}
	response.Ciphertext.C2.Add(hash.Mul(s.Curve.ModMul(response.Hash, s.SK[len(request.Ciphertexts)+1], s.Curve.GroupOrder)))
	return response, nil
}

// VerifyResponse returns a Pointcheval-Sanders signature if the BlindSingResponse is valid.
// Else, it returns error.
func (r *Recipient) VerifyResponse(response *BlindSignResponse) (*Signature, error) {
	sig := &Signature{}
	var err error
	// decrypt the ciphertext in the response
	// this results in Pointcheval-Sanders signature
	sig.S, err = r.EncSK.Decrypt(response.Ciphertext)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decrypt blind signature response")
	}
	sig.R = r.EncSK.Curve.HashToG1(r.Commitment.Bytes())

	// verify if the resulting signature is valid
	err = r.SignVerifier.Verify(append(r.Witness.messages, response.Hash), sig)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// Prove produces a serialized EncProof
func (r *Recipient) Prove() (*EncProof, error) {
	if len(r.Witness.messages) != len(r.Witness.encRandomness) {
		return nil, errors.Errorf("cannot generate encryption proof")
	}
	if len(r.Witness.messages) != len(r.PedersenParameters)-1 {
		return nil, errors.Errorf("cannot generate encryption proof")
	}
	if len(r.Ciphertexts) != len(r.Witness.messages) {
		return nil, errors.Errorf("cannot generate encryption proof")
	}
	rand, err := r.Curve.Rand()
	if err != nil {
		return nil, err
	}
	hash := r.Curve.HashToG1(r.Commitment.Bytes())

	// generate randomness
	randomness := &encProofRandomness{}
	randomness.comBlindingFactor = r.Curve.NewRandomZr(rand)
	for i := 0; i < len(r.Witness.messages); i++ {
		randomness.messages = append(randomness.messages, r.Curve.NewRandomZr(rand))
		randomness.encRandomness = append(randomness.encRandomness, r.Curve.NewRandomZr(rand))
	}
	// generate commitment for proof
	commitments := &EncProofCommitments{}
	for i := 0; i < len(r.Witness.messages); i++ {
		commitments.C1 = append(commitments.C1, r.EncSK.Gen.Mul(randomness.encRandomness[i]))
		T := hash.Mul(randomness.messages[i])
		T.Add(r.EncSK.H.Mul(randomness.encRandomness[i]))
		commitments.C2 = append(commitments.C2, T)
	}
	commitments.Commitment = r.PedersenParameters[len(r.PedersenParameters)-1].Mul(randomness.comBlindingFactor)
	for i := 0; i < len(r.PedersenParameters)-1; i++ {
		commitments.Commitment.Add(r.PedersenParameters[i].Mul(randomness.messages[i]))
	}

	proof := &EncProof{}
	// compute challenge
	var ciphertexts []*math.G1
	for i := 0; i < len(r.Ciphertexts); i++ {
		ciphertexts = append(ciphertexts, r.Ciphertexts[i].C1, r.Ciphertexts[i].C2)
	}

	bytes, err := common.GetG1Array(r.PedersenParameters, []*math.G1{r.EncSK.PublicKey.Gen, r.EncSK.PublicKey.H}, ciphertexts, []*math.G1{r.Commitment}, commitments.C1, commitments.C2, []*math.G1{commitments.Commitment}).Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate a blind signature request")
	}
	proof.Challenge = r.Curve.HashToZr(bytes)

	proof.Messages = make([]*math.Zr, len(r.Witness.messages))
	proof.EncRandomness = make([]*math.Zr, len(r.Witness.messages))
	// compute proof
	for i, m := range r.Witness.messages {
		proof.Messages[i] = r.Curve.ModAdd(randomness.messages[i], r.Curve.ModMul(m, proof.Challenge, r.Curve.GroupOrder), r.Curve.GroupOrder)
		proof.EncRandomness[i] = r.Curve.ModAdd(randomness.encRandomness[i], r.Curve.ModMul(r.Witness.encRandomness[i], proof.Challenge, r.Curve.GroupOrder), r.Curve.GroupOrder)
	}

	proof.ComBlindingFactor = r.Curve.ModAdd(randomness.comBlindingFactor, r.Curve.ModMul(r.Witness.comBlindingFactor, proof.Challenge, r.Curve.GroupOrder), r.Curve.GroupOrder)

	return proof, nil
}

// GenerateBlindSignRequest returns a blind Pointcheval-Sanders signature request
func (r *Recipient) GenerateBlindSignRequest() (*BlindSignRequest, error) {
	// encrypt
	r.Witness.encRandomness = make([]*math.Zr, len(r.Witness.messages))
	r.Ciphertexts = make([]*elgamal.Ciphertext, len(r.Witness.messages))
	hash := r.Curve.HashToG1(r.Commitment.Bytes())
	for i, m := range r.Witness.messages {
		var err error
		r.Ciphertexts[i], r.Witness.encRandomness[i], err = r.EncSK.PublicKey.Encrypt(hash.Mul(m))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate blind signature request")
		}
	}
	proof, err := r.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate blind signature request")
	}
	return &BlindSignRequest{Commitment: r.Commitment, Ciphertexts: r.Ciphertexts, Proof: proof, EncPK: r.EncSK.PublicKey}, nil
}

// Verify checks if EncProof is valid
func (v *encVerifier) Verify(p *EncProof) error {
	if len(v.Ciphertexts) != len(v.PedersenParameters)-1 {
		return errors.Errorf("failed to verify encryption proof: number of ciphertexts is different from [%d]", len(v.PedersenParameters)-1)
	}
	if len(v.Ciphertexts) != len(p.Messages) {
		return errors.Errorf("failed to verify encryption proof: number of proofs is different from [%d]", len(v.PedersenParameters)-1)
	}
	if len(v.Ciphertexts) != len(p.EncRandomness) {
		return errors.Errorf("failed to verify encryption proof: number of proofs is different from [%d]", len(v.PedersenParameters)-1)
	}
	if v.Commitment == nil {
		return errors.Errorf("failed to verify encryption proof: nil commitment")
	}
	hash := v.EncPK.Curve.HashToG1(v.Commitment.Bytes())

	commitments := &EncProofCommitments{}
	if v.PedersenParameters[len(v.PedersenParameters)-1] == nil {
		return errors.Errorf("failed to verify encryption proof: please initialize Pedersen generators")
	}
	if p.ComBlindingFactor == nil || p.Challenge == nil {
		return errors.Errorf("failed to verify encryption proof: nil proof element")
	}

	commitments.Commitment = v.PedersenParameters[len(v.PedersenParameters)-1].Mul(p.ComBlindingFactor)
	commitments.Commitment.Sub(v.Commitment.Mul(p.Challenge))
	for i := 0; i < len(v.PedersenParameters)-1; i++ {
		if v.PedersenParameters[i] == nil {
			return errors.Errorf("failed to verify encryption proof: please initialize Pedersen generators")
		}
		if p.Messages[i] == nil {
			return errors.Errorf("failed to verify encryption proof: nil proof element")
		}
		commitments.Commitment.Add(v.PedersenParameters[i].Mul(p.Messages[i]))
	}

	var ciphertexts []*math.G1
	for i := 0; i < len(v.Ciphertexts); i++ {
		if p.EncRandomness[i] == nil {
			return errors.Errorf("failed to verify encryption proof: nil proof element")
		}
		if v.Ciphertexts[i] == nil || v.Ciphertexts[i].C1 == nil || v.Ciphertexts[i].C2 == nil {
			return errors.Errorf("failed to verify encryption proof: nil ciphertexts")
		}

		T := v.EncPK.Gen.Mul(p.EncRandomness[i])
		T.Sub(v.Ciphertexts[i].C1.Mul(p.Challenge))

		commitments.C1 = append(commitments.C1, T)

		T = v.EncPK.H.Mul(p.EncRandomness[i])
		T.Add(hash.Mul(p.Messages[i]))
		T.Sub(v.Ciphertexts[i].C2.Mul(p.Challenge))

		commitments.C2 = append(commitments.C2, T)
		ciphertexts = append(ciphertexts, v.Ciphertexts[i].C1, v.Ciphertexts[i].C2)
	}
	// compute challenge
	raw, err := common.GetG1Array(v.PedersenParameters, []*math.G1{v.EncPK.Gen, v.EncPK.H}, ciphertexts, []*math.G1{v.Commitment}, commitments.C1, commitments.C2, []*math.G1{commitments.Commitment}).Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to verify encryption proof")
	}
	// check challenge
	if !v.Curve.HashToZr(raw).Equals(p.Challenge) {
		return errors.New("verification of encryption correctness failed")
	}
	return nil
}
