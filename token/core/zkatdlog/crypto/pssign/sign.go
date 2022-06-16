/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// Signer produces a Pointcheval-Sanders signature
type Signer struct {
	*SignVerifier
	SK []*math.Zr
}

// SignVerifier checks the validity of a Pointcheval-Sanders signature
type SignVerifier struct {
	PK    []*math.G2
	Q     *math.G2
	Curve *math.Curve
}

// Signature is a Pointcheval-Sanders signature
type Signature struct {
	R *math.G1
	S *math.G1
}

// Deserialize unmarshals a Pointcheval-Sanders signature
func (sig *Signature) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, sig)
}

// KeyGen instantiates Signer secret and public keys as a function
// of the length of the vector of messages to be signed
// KeyGen takes the length of vector of messages that Signer would like
// to sign
func (s *Signer) KeyGen(length int) error {
	s.SK = make([]*math.Zr, length+2)
	s.SignVerifier.PK = make([]*math.G2, length+2)
	// generate a random generator
	Q := s.Curve.GenG2
	rand, err := s.Curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	s.SignVerifier.Q = Q.Mul(s.Curve.NewRandomZr(rand))

	// generates secret keys
	for i := 0; i < length+2; i++ {
		s.SK[i] = s.Curve.NewRandomZr(rand)
		s.SignVerifier.PK[i] = s.SignVerifier.Q.Mul(s.SK[i])
	}
	return nil
}

// NewSigner returns a Signer as function of an array of secret keys
//and the corresponding array of public keys
func NewSigner(SK []*math.Zr, PK []*math.G2, Q *math.G2, c *math.Curve) *Signer {
	return &Signer{SK: SK, SignVerifier: NewVerifier(PK, Q, c)}
}

// NewVerifier returns a SignVerifier as a function of an array of public  kets
func NewVerifier(PK []*math.G2, Q *math.G2, c *math.Curve) *SignVerifier {
	return &SignVerifier{PK: PK, Q: Q, Curve: c}
}

// Sign returns a Pointcheval-Sanders signature for the vector of messages
// passed as argument
func (s *Signer) Sign(m []*math.Zr) (*Signature, error) {
	// check length of the vector to be signed
	if len(m) != len(s.SK)-2 {
		return nil, errors.Errorf("provide a message to sign of the right length")
	}

	// get random number generator
	rand, err := s.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	// initialize signature
	sig := &Signature{}
	sig.R = s.Curve.GenG1
	sig.R.Mul(s.Curve.NewRandomZr(rand)) // R is a random element in G1
	sig.S = s.Curve.NewG1()

	// compute S = R^{sk_0+ \sum m_i sk_i + m_{n+1} sk_{n+1}}
	// and m_{n+1} = H(m1, ..., m_n)
	sig.S.Add(sig.R.Mul(s.SK[0]))
	for i := 0; i < len(m); i++ {
		sig.S.Add(sig.R.Mul(s.Curve.ModMul(s.SK[1+i], m[i], s.Curve.GroupOrder)))
	}
	sig.S.Add(sig.R.Mul(s.Curve.ModMul(s.SK[len(m)+1], hashMessages(m, s.Curve), s.Curve.GroupOrder)))

	return sig, nil
}

// Verify takes a vector of messages and a signature, and validates it against
// SignVerifier
// Verify returns an error if the signature is invalid
// Verify checks if e(R, PK_0*\prod_{i=1}^n PK_i^{m_i}*PK_{n+1}^{m_{n+1}}) = e(S, Q)
func (v *SignVerifier) Verify(m []*math.Zr, sig *Signature) error {
	if len(m) != len(v.PK)-1 {
		return errors.New("PS signature cannot be verified!\n")
	}

	H := v.Curve.NewG2()
	for i := 0; i < len(m); i++ {
		H.Add(v.PK[1+i].Mul(m[i]))
	}
	H.Add(v.PK[0])

	T := v.Curve.NewG1()
	T.Sub(sig.S)

	e := v.Curve.Pairing2(v.Q, T, H, sig.R)
	e = v.Curve.FExp(e)

	if !e.IsUnity() {
		return errors.Errorf("invalid Pointcheval-Sanders signature")
	}
	return nil
}

// Randomize randomizes a Pointcheval-Sanders signature
func (v *SignVerifier) Randomize(sig *Signature) error {
	rand, err := v.Curve.Rand()
	if err != nil {
		return err
	}
	r := v.Curve.NewRandomZr(rand)
	// randomize signature
	sig.R = sig.R.Mul(r)
	sig.S = sig.S.Mul(r)

	return nil
}

// Copy copies a Pointcheval-Sanders signature
func (sig *Signature) Copy(sigma *Signature) {
	if sigma != nil {
		if sigma.S != nil && sigma.R != nil {
			sig.S = sigma.S.Copy()
			sig.R = sigma.R.Copy()
		}
	}
}

// Serialize marshals a Pointcheval-Sanders signature
func (sig *Signature) Serialize() ([]byte, error) {
	return json.Marshal(sig)
}

// hashMessages takes a vector of messages and returns
func hashMessages(m []*math.Zr, c *math.Curve) *math.Zr {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return c.HashToZr(bytesToHash)
}

// Serialize marshals the signer
func (s *Signer) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Signer) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, s)
}
