/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/pkg/errors"
)

type Signer struct {
	*SignVerifier
	SK []*bn256.Zr
}

type SignVerifier struct {
	PK []*bn256.G2
	Q  *bn256.G2
}

type Signature struct {
	R *bn256.G1
	S *bn256.G1
}

func (sig *Signature) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, sig)
}

func (s *Signer) KeyGen(length int) error {
	s.SK = make([]*bn256.Zr, length+2)
	s.SignVerifier = &SignVerifier{}
	s.SignVerifier.PK = make([]*bn256.G2, length+2)

	Q := bn256.G2Gen()
	rand, err := bn256.GetRand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	s.SignVerifier.Q = Q.Mul(bn256.RandModOrder(rand))

	for i := 0; i < length+2; i++ {
		s.SK[i] = bn256.RandModOrder(rand)
		s.SignVerifier.PK[i] = s.SignVerifier.Q.Mul(s.SK[i])
	}
	return nil
}

func NewSigner(SK []*bn256.Zr, PK []*bn256.G2, Q *bn256.G2) *Signer {
	return &Signer{SK: SK, SignVerifier: NewVerifier(PK, Q)}
}

func NewVerifier(PK []*bn256.G2, Q *bn256.G2) *SignVerifier {
	return &SignVerifier{PK: PK, Q: Q}
}

func (s *Signer) Sign(m []*bn256.Zr) (*Signature, error) {
	// check length of the vector to be signed
	if len(m) != len(s.SK)-2 {
		return nil, errors.Errorf("provide a message to sign of the right length")
	}

	// get RNG
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	// initialize signature
	sig := &Signature{}
	sig.R = bn256.G1Gen()
	sig.R.Mul(bn256.RandModOrder(rand)) // R is a random element in ECP
	sig.S = bn256.NewG1()

	// compute S
	sig.S.Add(sig.R.Mul(s.SK[0]))
	for i := 0; i < len(m); i++ {
		sig.S.Add(sig.R.Mul(bn256.ModMul(s.SK[1+i], m[i], bn256.Order)))
	}
	sig.S.Add(sig.R.Mul(bn256.ModMul(s.SK[len(m)+1], hashMessages(m), bn256.Order)))

	return sig, nil
}

func (v *SignVerifier) Verify(m []*bn256.Zr, sig *Signature) error {
	if len(m) != len(v.PK)-1 {
		return errors.New("PS signature cannot be verified!\n")
	}

	H := bn256.NewG2()
	for i := 0; i < len(m); i++ {
		H.Add(v.PK[1+i].Mul(m[i]))
	}
	H.Add(v.PK[0])

	T := bn256.NewG1()
	T.Sub(sig.S)

	e := bn256.Pairing(v.Q, T, H, sig.R)
	e = bn256.FinalExp(e)

	if !e.IsUnity() {
		return errors.Errorf("invalid Pointcheval-Sanders signature")
	}
	return nil
}

func (sig *Signature) Randomize() error {
	rand, err := bn256.GetRand()
	if err != nil {
		return err
	}
	r := bn256.RandModOrder(rand)
	// randomize signature
	sig.R = sig.R.Mul(r)
	sig.S = sig.S.Mul(r)

	return nil
}

func (sig *Signature) Copy(sigma *Signature) {
	sig.S = bn256.NewG1()
	sig.S.Copy(sigma.S)
	sig.R = bn256.NewG1()
	sig.R.Copy(sigma.R)
}

func (sig *Signature) Serialize() ([]byte, error) {
	return json.Marshal(sig)
}

func hashMessages(m []*bn256.Zr) *bn256.Zr {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return bn256.HashModOrder(bytesToHash)
}

func (s *Signer) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Signer) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, s)
}
