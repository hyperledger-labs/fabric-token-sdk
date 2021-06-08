/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/api"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/pkg/errors"
)

// this implements signing identity
type NYMSigner struct {
	*NYMVerifier
	SK *bn256.Zr
	BF *bn256.Zr
}

// get verifier
func (s *NYMSigner) GetPublicVersion() api.Identity {
	return s.NYMVerifier
}

// sign message anonymously using Schnorr signature
func (s *NYMSigner) Sign(message []byte) ([]byte, error) {
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, err
	}
	skRandomness := bn256.RandModOrder(rand)
	bfRandomness := bn256.RandModOrder(rand)

	com := s.NYMParams[0].Mul(skRandomness)
	com.Add(s.NYMParams[1].Mul(bfRandomness))

	sig := &NYMSig{}
	sig.Challenge = bn256.HashModOrder(append(message, GetG1Array(s.NYMParams, []*bn256.G1{s.NYM, com}).Bytes()...))
	sig.SK = bn256.ModMul(sig.Challenge, s.SK, bn256.Order)
	sig.SK = bn256.ModAdd(sig.SK, skRandomness, bn256.Order)

	sig.BF = bn256.ModMul(sig.Challenge, s.BF, bn256.Order)
	sig.BF = bn256.ModAdd(sig.BF, bfRandomness, bn256.Order)

	bytes, err := sig.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to serialized nym signature")
	}
	return bytes, nil
}

// this implements sig verifier
type NYMVerifier struct {
	NYMParams []*bn256.G1
	NYM       *bn256.G1
}

// return serialized pseudonym
func (v *NYMVerifier) Serialize() ([]byte, error) {
	bytes := v.NYM.Bytes()
	return bytes, nil
}

// return serialized pseudonym
func (v *NYMVerifier) Deserialize(raw []byte) error {
	var err error
	v.NYM, err = bn256.NewG1FromBytes(raw)
	return err
}

// verify signature relative to pseudonym
func (v *NYMVerifier) Verify(message []byte, signature []byte) error {
	sig := &NYMSig{}
	err := sig.Deserialize(signature)
	if err != nil {
		return errors.Errorf("failed to deserialize nym signature")
	}

	sv := &SchnorrVerifier{PedParams: v.NYMParams}
	sp := &SchnorrProof{Challenge: sig.Challenge, Proof: []*bn256.Zr{sig.SK, sig.BF}, Statement: v.NYM}
	com := sv.RecomputeCommitment(sp)
	chal := bn256.HashModOrder(append(message, GetG1Array(v.NYMParams, []*bn256.G1{v.NYM, com}).Bytes()...))
	if chal.Cmp(sig.Challenge) != 0 {
		return errors.Errorf("invalid nym signature")
	}
	return nil
}

// Pseudonyms signature (schnorr)
type NYMSig struct {
	SK        *bn256.Zr
	BF        *bn256.Zr
	Challenge *bn256.Zr
}

// Intermediate struct for serialization and deserialization
type NYMSigBytes struct {
	SK        []byte
	BF        []byte
	Challenge []byte
}

func (s *NYMSig) Serialize() ([]byte, error) {
	pb := &NYMSigBytes{}

	pb.SK = s.SK.Bytes()

	pb.BF = s.BF.Bytes()

	pb.Challenge = s.Challenge.Bytes()

	return json.Marshal(pb)
}

func (s *NYMSig) Deserialize(raw []byte) error {
	pb := &NYMSigBytes{}
	err := json.Unmarshal(raw, pb)
	if err != nil {
		return err
	}
	s.SK = bn256.NewZrFromBytes(pb.SK)
	s.Challenge = bn256.NewZrFromBytes(pb.Challenge)
	s.BF = bn256.NewZrFromBytes(pb.BF)

	return nil
}
