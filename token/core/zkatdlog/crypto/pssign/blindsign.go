/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/elgamal"
	"github.com/pkg/errors"
)

type Recipient struct {
	*EncVerifier
	*SignVerifier
	EncSK   *elgamal.SecretKey
	Witness *EncWitness
	Curve   *math.Curve
}

type BlindSigner struct {
	*Signer
	PedersenParameters []*math.G1
}

func NewBlindSigner(SK []*math.Zr, PK []*math.G2, Q *math.G2, pp []*math.G1, curve *math.Curve) *BlindSigner {
	s := &BlindSigner{PedersenParameters: pp}
	s.Signer = NewSigner(SK, PK, Q, curve)
	return s
}

func NewRecipient(messages []*math.Zr, blindingfactor *math.Zr, com *math.G1, sk *math.Zr, gen, pk *math.G1, pp []*math.G1, PK []*math.G2, Q *math.G2, curve *math.Curve) *Recipient {

	return &Recipient{
		Witness: &EncWitness{
			messages:          messages,
			comBlindingFactor: blindingfactor,
		},
		EncSK: elgamal.NewSecretKey(sk, gen, pk, curve),
		EncVerifier: &EncVerifier{
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

type EncVerifier struct {
	PedersenParameters []*math.G1            //g_0, g_1, g_2, g_3, h (owner, type, value, sn, randomness)
	Commitment         *math.G1              // commitment to messages
	Ciphertexts        []*elgamal.Ciphertext // encryption of messages
	EncPK              *elgamal.PublicKey
	Curve              *math.Curve
}

type EncWitness struct {
	messages          []*math.Zr // messages
	encRandomness     []*math.Zr // randomness used in encryption
	comBlindingFactor *math.Zr   // randomness used in commitment
}

type EncProof struct {
	Messages          []*math.Zr
	EncRandomness     []*math.Zr
	ComBlindingFactor *math.Zr
	Challenge         *math.Zr
}

type BlindSignRequest struct {
	Commitment  *math.G1
	Ciphertexts []*elgamal.Ciphertext
	Proof       []byte
	EncPK       *elgamal.PublicKey
}

type BlindSignResponse struct {
	Hash       *math.Zr
	Ciphertext *elgamal.Ciphertext
}

type encProofRandomness struct {
	messages          []*math.Zr
	encRandomness     []*math.Zr
	comBlindingFactor *math.Zr
}

type EncProofCommitments struct {
	C1         []*math.G1
	C2         []*math.G1
	Commitment *math.G1
}

func (s *BlindSigner) BlindSign(request *BlindSignRequest) (*BlindSignResponse, error) {
	if len(request.Ciphertexts) != len(s.PK)-2 {
		return nil, errors.Errorf("number of ciphertexts in blind signature request does not match number of public keys: expect [%d], got [%d]", len(s.PK)-2, len(request.Ciphertexts))
	}
	v := &EncVerifier{Commitment: request.Commitment, Ciphertexts: request.Ciphertexts, EncPK: request.EncPK, PedersenParameters: s.PedersenParameters, Curve: s.Curve}
	err := v.Verify(request.Proof)
	if err != nil {
		return nil, err
	}
	response := &BlindSignResponse{Hash: s.Curve.HashToZr(request.Proof)}
	response.Ciphertext = &elgamal.Ciphertext{}
	hash := s.Signer.Curve.HashToG1(request.Commitment.Bytes())
	response.Ciphertext.C1 = s.Curve.NewG1()
	response.Ciphertext.C2 = hash.Mul(s.SK[0])
	for i := 0; i < len(request.Ciphertexts); i++ {
		response.Ciphertext.C1.Add(request.Ciphertexts[i].C1.Mul(s.SK[i+1]))
		response.Ciphertext.C2.Add(request.Ciphertexts[i].C2.Mul(s.SK[i+1]))
	}
	response.Ciphertext.C2.Add(hash.Mul(s.Curve.ModMul(response.Hash, s.SK[len(request.Ciphertexts)+1], s.Curve.GroupOrder)))
	return response, nil
}

func (r *Recipient) VerifyResponse(response *BlindSignResponse) (*Signature, error) {
	sig := &Signature{}
	sig.S = r.EncSK.Decrypt(response.Ciphertext)
	var err error
	sig.R = r.EncSK.Curve.HashToG1(r.Commitment.Bytes())

	err = r.SignVerifier.Verify(append(r.Witness.messages, response.Hash), sig)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (r *Recipient) Prove() ([]byte, error) {
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
	sv := &common.SchnorrVerifier{Curve: r.Curve}
	proof.Challenge = sv.ComputeChallenge(common.GetG1Array(r.PedersenParameters, []*math.G1{r.EncSK.PublicKey.Gen, r.EncSK.PublicKey.H}, ciphertexts, []*math.G1{r.Commitment}, commitments.C1, commitments.C2, []*math.G1{commitments.Commitment}))

	proof.Messages = make([]*math.Zr, len(r.Witness.messages))
	proof.EncRandomness = make([]*math.Zr, len(r.Witness.messages))
	// generate proof
	for i, m := range r.Witness.messages {
		proof.Messages[i] = r.Curve.ModAdd(randomness.messages[i], r.Curve.ModMul(m, proof.Challenge, r.Curve.GroupOrder), r.Curve.GroupOrder)
		proof.EncRandomness[i] = r.Curve.ModAdd(randomness.encRandomness[i], r.Curve.ModMul(r.Witness.encRandomness[i], proof.Challenge, r.Curve.GroupOrder), r.Curve.GroupOrder)
	}

	proof.ComBlindingFactor = r.Curve.ModAdd(randomness.comBlindingFactor, r.Curve.ModMul(r.Witness.comBlindingFactor, proof.Challenge, r.Curve.GroupOrder), r.Curve.GroupOrder)

	return json.Marshal(proof)
}

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

func (v *EncVerifier) Verify(proof []byte) error {
	if len(v.Ciphertexts) != len(v.PedersenParameters)-1 {
		return errors.Errorf("failed to verify encryption proof: number of ciphertexts is different from [%d]", len(v.PedersenParameters)-1)
	}

	p := &EncProof{}
	err := json.Unmarshal(proof, p)
	if err != nil {
		return errors.Errorf("failed to unmarshal encryption proof")
	}
	if len(v.Ciphertexts) != len(p.Messages) {
		return errors.Errorf("failed to verify encryption proof: number of proofs is different from [%d]", len(v.PedersenParameters)-1)
	}

	if len(v.Ciphertexts) != len(p.EncRandomness) {
		return errors.Errorf("failed to verify encryption proof: number of proofs is different from [%d]", len(v.PedersenParameters)-1)
	}

	hash := v.EncPK.Curve.HashToG1(v.Commitment.Bytes())
	commitments := &EncProofCommitments{}

	commitments.Commitment = v.PedersenParameters[len(v.PedersenParameters)-1].Mul(p.ComBlindingFactor)
	commitments.Commitment.Sub(v.Commitment.Mul(p.Challenge))
	for i := 0; i < len(v.PedersenParameters)-1; i++ {
		commitments.Commitment.Add(v.PedersenParameters[i].Mul(p.Messages[i]))
	}

	var ciphertexts []*math.G1
	for i := 0; i < len(v.Ciphertexts); i++ {
		T := v.EncPK.Gen.Mul(p.EncRandomness[i])
		T.Sub(v.Ciphertexts[i].C1.Mul(p.Challenge))
		commitments.C1 = append(commitments.C1, T)
		T = v.EncPK.H.Mul(p.EncRandomness[i])
		T.Add(hash.Mul(p.Messages[i]))
		T.Sub(v.Ciphertexts[i].C2.Mul(p.Challenge))
		commitments.C2 = append(commitments.C2, T)
		ciphertexts = append(ciphertexts, v.Ciphertexts[i].C1, v.Ciphertexts[i].C2)
	}
	sv := &common.SchnorrVerifier{Curve: v.Curve}
	// compute challenge
	chal := sv.ComputeChallenge(common.GetG1Array(v.PedersenParameters, []*math.G1{v.EncPK.Gen, v.EncPK.H}, ciphertexts, []*math.G1{v.Commitment}, commitments.C1, commitments.C2, []*math.G1{commitments.Commitment}))
	// check challenge
	if !chal.Equals(p.Challenge) {
		return errors.Errorf("verification of encryption correctness failed")
	}
	return nil
}
