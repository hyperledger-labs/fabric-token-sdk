/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// nymSigner implements signing identity
// This is a signer for the Signature of Knowledge of (SK, BF) such that NYM = PedGen^SK*Q^BF
type nymSigner struct {
	*NymVerifier
	SK *math.Zr
	BF *math.Zr
}

func NewNymSigner(sk, bf *math.Zr, nymParams []*math.G1, nym *math.G1, curve *math.Curve) (*nymSigner, error) {
	if sk == nil || bf == nil {
		return nil, errors.New("failed to initialize nym signer: please provide non-nil parameters")
	}
	v, err := NewNymVerifier(nymParams, nym, curve)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize nym signer")
	}
	return &nymSigner{NymVerifier: v, SK: sk, BF: bf}, nil

}

func NewNymVerifier(nymParams []*math.G1, nym *math.G1, curve *math.Curve) (*NymVerifier, error) {
	if nym == nil || curve == nil {
		return nil, errors.New("failed to initialize nym verifier: please provide non-nil parameters")
	}
	if len(nymParams) != 2 || nymParams[0] == nil || nymParams[1] == nil {
		return nil, errors.New("failed to initialize nym verifier: invalid commitment parameters")
	}
	return &NymVerifier{NYMParams: nymParams, NYM: nym, Curve: curve}, nil
}

// Sign produces a signature of knowledge (SOK for short) for (SK, BF) such that NYM = PedGen^SK*Q^BF
func (s *nymSigner) Sign(message []byte) ([]byte, error) {
	if message == nil {
		return nil, errors.New("failed to sign anonymously: nil message")
	}
	// prepare SOK commitments
	rand, err := s.Curve.Rand()
	if err != nil {
		return nil, err
	}
	skRandomness := s.Curve.NewRandomZr(rand)
	bfRandomness := s.Curve.NewRandomZr(rand)

	com := s.NYMParams[0].Mul(skRandomness)
	com.Add(s.NYMParams[1].Mul(bfRandomness))

	// prepare NYM signature
	sig := &NYMSig{}
	raw, err := GetG1Array(s.NYMParams, []*math.G1{s.NYM, com}).Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sign with nym signer")
	}

	// compute challenge
	sig.Challenge = s.Curve.HashToZr(append(message, raw...))

	if s.Curve.GroupOrder == nil {
		return nil, errors.New("failed to sign with nym signer: nil group order")
	}
	// compute ZKPs
	sig.SK = s.Curve.ModMul(sig.Challenge, s.SK, s.Curve.GroupOrder)
	sig.SK = s.Curve.ModAdd(sig.SK, skRandomness, s.Curve.GroupOrder)

	sig.BF = s.Curve.ModMul(sig.Challenge, s.BF, s.Curve.GroupOrder)
	sig.BF = s.Curve.ModAdd(sig.BF, bfRandomness, s.Curve.GroupOrder)
	// serialize SOK
	bytes, err := sig.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to serialize nym signature")
	}
	return bytes, nil
}

// NymVerifier verify SOK signature produced by nymSigner
type NymVerifier struct {
	NYMParams []*math.G1
	NYM       *math.G1
	Curve     *math.Curve
}

// Serialize returns a serialized pseudonym (PedGen^SK*Q^BF)
func (v *NymVerifier) Serialize() ([]byte, error) {
	if v.NYM == nil {
		return nil, errors.New("failed to serialize NymVerifier: a nil pseudonym")
	}
	bytes := v.NYM.Bytes()
	return bytes, nil
}

// Deserialize returns an error if it fails to deserialize a NymVerifier from raw.
func (v *NymVerifier) Deserialize(raw []byte) error {
	var err error
	if v.Curve == nil {
		return errors.New("failed to deserialize pseudonym: please initialize curve for NymVerifier")
	}
	v.NYM, err = v.Curve.NewG1FromBytes(raw)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize pseudonym")
	}
	return nil
}

// Verify SOK signature relative to pseudonym in NymVerifier
func (v *NymVerifier) Verify(message []byte, signature []byte) error {
	sig := &NYMSig{}
	err := sig.Deserialize(signature)
	if err != nil {
		return errors.Errorf("failed to deserialize nym signature")
	}

	// initialize Schnorr verifier
	sv := &SchnorrVerifier{PedParams: v.NYMParams}
	sp := &SchnorrProof{Challenge: sig.Challenge, Proof: []*math.Zr{sig.SK, sig.BF}, Statement: v.NYM}
	// recompute commitments
	com, err := sv.RecomputeCommitment(sp)
	if err != nil {
		return errors.Errorf("failed to verify nym signature")
	}
	raw, err := GetG1Array(v.NYMParams, []*math.G1{v.NYM, com}).Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to verify nym signature")
	}
	// compute challenge
	if v.Curve == nil {
		errors.Errorf("failed to verify nym signature: please initialize curve")
	}
	chal := v.Curve.HashToZr(append(message, raw...))
	// check challenge equality
	if sig.Challenge == nil {
		return errors.Errorf("failed verify nym signature: challenge in signature is nil")
	}
	if !chal.Equals(sig.Challenge) {
		return errors.Errorf("invalid nym signature")
	}
	return nil
}

// NYMSig is the SOK signature produced by nymSigner
type NYMSig struct {
	SK        *math.Zr
	BF        *math.Zr
	Challenge *math.Zr
}

// Serialize marshals NYMSig
func (s *NYMSig) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

// Deserialize un-marshals NYMSig
func (s *NYMSig) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, s)
}
