/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/elgamal"
	"github.com/pkg/errors"
)

type Recipient struct {
	*EncVerifier
	*SignVerifier
	EncSK   *elgamal.SecretKey
	Witness *EncWitness
}

type BlindSigner struct {
	*Signer
	PedersenParameters []*bn256.G1
}

func NewBlindSigner(SK []*bn256.Zr, PK []*bn256.G2, Q *bn256.G2, pp []*bn256.G1) *BlindSigner {
	s := &BlindSigner{PedersenParameters: pp}
	s.Signer = NewSigner(SK, PK, Q)
	return s
}

func NewRecipient(messages []*bn256.Zr, blindingfactor *bn256.Zr, com *bn256.G1, sk *bn256.Zr, gen, pk *bn256.G1, pp []*bn256.G1, PK []*bn256.G2, Q *bn256.G2) *Recipient {

	return &Recipient{
		Witness: &EncWitness{
			messages:          messages,
			comBlindingFactor: blindingfactor,
		},
		EncSK: elgamal.NewSecretKey(sk, gen, pk),
		EncVerifier: &EncVerifier{
			PedersenParameters: pp,
			Commitment:         com,
		},
		SignVerifier: &SignVerifier{
			PK: PK,
			Q:  Q,
		},
	}
}

type EncVerifier struct {
	PedersenParameters []*bn256.G1           //g_0, g_1, g_2, g_3, h (owner, type, value, sn, randomness)
	Commitment         *bn256.G1             // commitment to messages
	Ciphertexts        []*elgamal.Ciphertext // encryption of messages
	EncPK              *elgamal.PublicKey
}

type EncWitness struct {
	messages          []*bn256.Zr // messages
	encRandomness     []*bn256.Zr // randomness used in encryption
	comBlindingFactor *bn256.Zr   // randomness used in commitment
}

type EncProof struct {
	Messages          []*bn256.Zr
	EncRandomness     []*bn256.Zr
	ComBlindingFactor *bn256.Zr
	Challenge         *bn256.Zr
}

type BlindSignRequest struct {
	Commitment  *bn256.G1
	Ciphertexts []*elgamal.Ciphertext
	Proof       []byte
	EncPK       *elgamal.PublicKey
}

type BlindSignResponse struct {
	Hash       *bn256.Zr
	Ciphertext *elgamal.Ciphertext
}

type encProofRandomness struct {
	messages          []*bn256.Zr
	encRandomness     []*bn256.Zr
	comBlindingFactor *bn256.Zr
}

type EncProofCommitments struct {
	C1         []*bn256.G1
	C2         []*bn256.G1
	Commitment *bn256.G1
}

func (s *BlindSigner) BlindSign(request *BlindSignRequest) (*BlindSignResponse, error) {
	if len(request.Ciphertexts) != len(s.PK)-2 {
		return nil, errors.Errorf("number of ciphertexts in blind signature request does not match number of public keys: expect [%d], got [%d]", len(s.PK)-2, len(request.Ciphertexts))
	}
	v := &EncVerifier{Commitment: request.Commitment, Ciphertexts: request.Ciphertexts, EncPK: request.EncPK, PedersenParameters: s.PedersenParameters}
	hash, err := bn256.HashToG1(request.Commitment.Bytes())
	if err != nil {
		return nil, errors.Errorf("failed to hash commitment")
	}
	err = v.Verify(request.Proof)
	if err != nil {
		return nil, err
	}
	response := &BlindSignResponse{Hash: bn256.HashModOrder(request.Proof)}
	response.Ciphertext = &elgamal.Ciphertext{}

	response.Ciphertext.C1 = bn256.NewG1()
	response.Ciphertext.C2 = hash.Mul(s.SK[0])
	for i := 0; i < len(request.Ciphertexts); i++ {
		response.Ciphertext.C1.Add(request.Ciphertexts[i].C1.Mul(s.SK[i+1]))
		response.Ciphertext.C2.Add(request.Ciphertexts[i].C2.Mul(s.SK[i+1]))
	}
	response.Ciphertext.C2.Add(hash.Mul(bn256.ModMul(response.Hash, s.SK[len(request.Ciphertexts)+1], bn256.Order)))
	return response, nil
}

func (r *Recipient) VerifyResponse(response *BlindSignResponse) (*Signature, error) {
	sig := &Signature{}
	sig.S = r.EncSK.Decrypt(response.Ciphertext)
	var err error
	sig.R, err = bn256.HashToG1(r.Commitment.Bytes())
	if err != nil {
		return nil, errors.Errorf("failed to hash commitment")
	}

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
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, err
	}
	hash, err := bn256.HashToG1(r.Commitment.Bytes())
	if err != nil {
		return nil, errors.Errorf("failed to hash commitment")
	}
	// generate randomness
	randomness := &encProofRandomness{}
	randomness.comBlindingFactor = bn256.RandModOrder(rand)
	for i := 0; i < len(r.Witness.messages); i++ {
		randomness.messages = append(randomness.messages, bn256.RandModOrder(rand))
		randomness.encRandomness = append(randomness.encRandomness, bn256.RandModOrder(rand))
	}
	// generate commitment for proof
	commitments := &EncProofCommitments{}
	for i := 0; i < len(r.Witness.messages); i++ {
		commitments.C1 = append(commitments.C1, r.EncSK.Gen.Mul(randomness.encRandomness[i]))
		commitments.C2 = append(commitments.C2, hash.Mul(randomness.messages[i]).Add(r.EncSK.H.Mul(randomness.encRandomness[i])))
	}
	commitments.Commitment = r.PedersenParameters[len(r.PedersenParameters)-1].Mul(randomness.comBlindingFactor)
	for i := 0; i < len(r.PedersenParameters)-1; i++ {
		commitments.Commitment.Add(r.PedersenParameters[i].Mul(randomness.messages[i]))
	}

	proof := &EncProof{}
	// compute challenge
	var ciphertexts []*bn256.G1
	for i := 0; i < len(r.Ciphertexts); i++ {
		ciphertexts = append(ciphertexts, r.Ciphertexts[i].C1, r.Ciphertexts[i].C2)
	}
	proof.Challenge = common.ComputeChallenge(common.GetG1Array(r.PedersenParameters, []*bn256.G1{r.EncSK.PublicKey.Gen, r.EncSK.PublicKey.H}, ciphertexts, []*bn256.G1{r.Commitment}, commitments.C1, commitments.C2, []*bn256.G1{commitments.Commitment}))

	proof.Messages = make([]*bn256.Zr, len(r.Witness.messages))
	proof.EncRandomness = make([]*bn256.Zr, len(r.Witness.messages))
	// generate proof
	for i, m := range r.Witness.messages {
		proof.Messages[i] = bn256.ModAdd(randomness.messages[i], bn256.ModMul(m, proof.Challenge, bn256.Order), bn256.Order)
		proof.EncRandomness[i] = bn256.ModAdd(randomness.encRandomness[i], bn256.ModMul(r.Witness.encRandomness[i], proof.Challenge, bn256.Order), bn256.Order)
	}

	proof.ComBlindingFactor = bn256.ModAdd(randomness.comBlindingFactor, bn256.ModMul(r.Witness.comBlindingFactor, proof.Challenge, bn256.Order), bn256.Order)

	return json.Marshal(proof)
}

func (r *Recipient) GenerateBlindSignRequest() (*BlindSignRequest, error) {
	// encrypt
	r.Witness.encRandomness = make([]*bn256.Zr, len(r.Witness.messages))
	r.Ciphertexts = make([]*elgamal.Ciphertext, len(r.Witness.messages))
	hash, err := bn256.HashToG1(r.Commitment.Bytes())
	if err != nil {
		return nil, errors.Errorf("failed to hash commitment")
	}
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

	hash, err := bn256.HashToG1(v.Commitment.Bytes())
	if err != nil {
		return errors.Errorf("failed to hash commitment")
	}
	commitments := &EncProofCommitments{}

	commitments.Commitment = v.PedersenParameters[len(v.PedersenParameters)-1].Mul(p.ComBlindingFactor).Sub(v.Commitment.Mul(p.Challenge))
	for i := 0; i < len(v.PedersenParameters)-1; i++ {
		commitments.Commitment.Add(v.PedersenParameters[i].Mul(p.Messages[i]))
	}

	var ciphertexts []*bn256.G1
	for i := 0; i < len(v.Ciphertexts); i++ {
		commitments.C1 = append(commitments.C1, v.EncPK.Gen.Mul(p.EncRandomness[i]).Sub(v.Ciphertexts[i].C1.Mul(p.Challenge)))
		commitments.C2 = append(commitments.C2, v.EncPK.H.Mul(p.EncRandomness[i]).Add(hash.Mul(p.Messages[i])).Sub(v.Ciphertexts[i].C2.Mul(p.Challenge)))
		ciphertexts = append(ciphertexts, v.Ciphertexts[i].C1, v.Ciphertexts[i].C2)
	}
	// compute challenge
	chal := common.ComputeChallenge(common.GetG1Array(v.PedersenParameters, []*bn256.G1{v.EncPK.Gen, v.EncPK.H}, ciphertexts, []*bn256.G1{v.Commitment}, commitments.C1, commitments.C2, []*bn256.G1{commitments.Commitment}))
	// check challenge
	if chal.Cmp(p.Challenge) != 0 {
		return errors.Errorf("verification of encryption correctness failed")
	}
	return nil
}
