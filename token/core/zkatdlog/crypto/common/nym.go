/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// this implements signing identity
type NYMSigner struct {
	*NYMVerifier
	SK *math.Zr
	BF *math.Zr
}

// get verifier
func (s *NYMSigner) Serialize() ([]byte, error) {
	return s.NYMVerifier.Serialize()
}

// sign message anonymously using Schnorr signature
func (s *NYMSigner) Sign(message []byte) ([]byte, error) {
	rand, err := s.Curve.Rand()
	if err != nil {
		return nil, err
	}
	skRandomness := s.Curve.NewRandomZr(rand)
	bfRandomness := s.Curve.NewRandomZr(rand)

	com := s.NYMParams[0].Mul(skRandomness)
	com.Add(s.NYMParams[1].Mul(bfRandomness))

	sig := &NYMSig{}
	sig.Challenge = s.Curve.HashToZr(append(message, GetG1Array(s.NYMParams, []*math.G1{s.NYM, com}).Bytes()...))
	sig.SK = s.Curve.ModMul(sig.Challenge, s.SK, s.Curve.GroupOrder)
	sig.SK = s.Curve.ModAdd(sig.SK, skRandomness, s.Curve.GroupOrder)

	sig.BF = s.Curve.ModMul(sig.Challenge, s.BF, s.Curve.GroupOrder)
	sig.BF = s.Curve.ModAdd(sig.BF, bfRandomness, s.Curve.GroupOrder)

	bytes, err := sig.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to serialized nym signature")
	}
	return bytes, nil
}

// this implements sig verifier
type NYMVerifier struct {
	NYMParams []*math.G1
	NYM       *math.G1
	Curve     *math.Curve
}

// return serialized pseudonym
func (v *NYMVerifier) Serialize() ([]byte, error) {
	bytes := v.NYM.Bytes()
	return bytes, nil
}

// return serialized pseudonym
func (v *NYMVerifier) Deserialize(raw []byte) error {
	var err error
	v.NYM, err = v.Curve.NewG1FromBytes(raw)
	return err
}

// verify signature relative to pseudonym
func (v *NYMVerifier) Verify(message []byte, signature []byte) error {
	sig, err := v.DeserializeSignature(signature)
	if err != nil {
		return errors.Errorf("failed to deserialize nym signature")
	}

	sv := &SchnorrVerifier{PedParams: v.NYMParams} // todo Curve?
	sp := &SchnorrProof{Challenge: sig.Challenge, Proof: []*math.Zr{sig.SK, sig.BF}, Statement: v.NYM}
	com := sv.RecomputeCommitment(sp)
	chal := v.Curve.HashToZr(append(message, GetG1Array(v.NYMParams, []*math.G1{v.NYM, com}).Bytes()...))
	if !chal.Equals(sig.Challenge) {
		return errors.Errorf("invalid nym signature")
	}
	return nil
}

// Pseudonyms signature (schnorr)
type NYMSig struct {
	SK        *math.Zr
	BF        *math.Zr
	Challenge *math.Zr
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

func (v *NYMVerifier) DeserializeSignature(raw []byte) (*NYMSig, error) {
	pb := &NYMSigBytes{}
	err := json.Unmarshal(raw, pb)
	if err != nil {
		return nil, err
	}
	s := &NYMSig{}
	s.SK = v.Curve.NewZrFromBytes(pb.SK)
	s.Challenge = v.Curve.NewZrFromBytes(pb.Challenge)
	s.BF = v.Curve.NewZrFromBytes(pb.BF)

	return s, nil
}
