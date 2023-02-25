/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
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
	if s.SignVerifier == nil {
		return errors.New("cannot generate Pointcheval-Sanders signature keys: SignVerifier is nil, " +
			"please initialize signer properly")
	}
	s.SignVerifier.PK = make([]*math.G2, length+2)
	// generate a random generator
	if s.Curve == nil || s.Curve.GenG2 == nil {
		return errors.New("cannot generate Pointcheval-Sanders signature keys: please initialize curve properly")
	}
	rand, err := s.Curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	s.SignVerifier.Q = s.Curve.GenG2.Mul(s.Curve.NewRandomZr(rand))

	// generates secret keys
	for i := 0; i < length+2; i++ {
		s.SK[i] = s.Curve.NewRandomZr(rand)
		s.SignVerifier.PK[i] = s.SignVerifier.Q.Mul(s.SK[i])
	}
	return nil
}

// NewSigner returns a Signer as function of an array of secret keys
// and the corresponding array of public keys
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
		return nil, errors.New("cannot produce a Pointcheval-Sanders signature: please provide a vector of messages to sign of the right length")
	}
	if s.Curve == nil || s.Curve.GroupOrder == nil || s.Curve.GenG1 == nil {
		return nil, errors.New("cannot produce a Pointcheval-Sanders signature: please initialize curve properly")
	}
	// get random number generator
	rand, err := s.Curve.Rand()
	if err != nil {
		return nil, errors.Wrap(err, "cannot produce a Pointcheval-Sanders signature")
	}

	// initialize signature
	sig := &Signature{}
	sig.R = s.Curve.GenG1
	sig.R.Mul(s.Curve.NewRandomZr(rand)) // R is a random element in G1
	sig.S = s.Curve.NewG1()

	// compute S = R^{sk_0+ \sum m_i sk_i + m_{n+1} sk_{n+1}}
	// and m_{n+1} = H(m1, ..., m_n)
	if s.SK[0] == nil {
		return nil, errors.Errorf("cannot produce a Pointcheval-Sanders signature: SK[%d] is nil", 0)
	}
	sig.S.Add(sig.R.Mul(s.SK[0]))
	for i := 0; i < len(m); i++ {
		if s.SK[1+i] == nil || m[i] == nil {
			return nil, errors.Errorf("cannot sign m[%d] using Pointcheval-Sanders signature: either m[%d] or SK[%d+1] is nil", i, i, i)
		}
		sig.S.Add(sig.R.Mul(s.Curve.ModMul(s.SK[1+i], m[i], s.Curve.GroupOrder)))
	}
	if s.SK[1+len(m)] == nil {
		return nil, errors.Errorf("cannot produce a Pointcheval-Sanders signature: SK[%d] is nil", 1+len(m))
	}
	sig.S.Add(sig.R.Mul(s.Curve.ModMul(s.SK[len(m)+1], hashMessages(m, s.Curve), s.Curve.GroupOrder)))

	return sig, nil
}

// Verify takes a vector of messages and a signature, and validates it against
// SignVerifier
// Verify returns an error if the signature is invalid
// Verify checks if e(R, PK_0*\prod_{i=1}^n PK_i^{m_i}*PK_{n+1}^{m_{n+1}}) = e(S, Q)
func (v *SignVerifier) Verify(m []*math.Zr, sig *Signature) error {
	if sig == nil || sig.S == nil || sig.R == nil {
		return errors.New("cannot verify Pointcheval-Sanders signature: nil signature")
	}
	if v.Curve == nil {
		return errors.New("cannot verify Pointcheval-Sanders signature: please initialize curve properly")
	}
	if len(m) != len(v.PK)-1 {
		return errors.New("cannot verify Pointcheval-Sanders signature: length of vector of messages does not match the length of the public  key")
	}

	H := v.Curve.NewG2()
	if v.PK[0] == nil {
		return errors.New("cannot verify Pointcheval-Sanders signature : PK[0] is nil")
	}
	for i := 0; i < len(m); i++ {
		if v.PK[1+i] == nil || m[i] == nil {
			return errors.Errorf("cannot verify Pointcheval-Sanders signature : m[%d] or PK[1+%d] is nil", i, i)
		}
		H.Add(v.PK[1+i].Mul(m[i]))
	}
	H.Add(v.PK[0])

	T := v.Curve.NewG1()
	T.Sub(sig.S)
	if v.Q == nil {
		return errors.New("cannot verify Pointcheval-Sanders signature : generator Q is nil")
	}
	e := v.Curve.Pairing2(v.Q, T, H, sig.R)
	e = v.Curve.FExp(e)

	if !e.IsUnity() {
		return errors.Errorf("invalid Pointcheval-Sanders signature")
	}
	return nil
}

// Randomize randomizes a Pointcheval-Sanders signature
func (v *SignVerifier) Randomize(sig *Signature) error {
	if sig == nil || sig.S == nil || sig.R == nil {
		return errors.New("cannot randomize Pointcheval-Sanders signature: nil signature")
	}
	if v.Curve == nil {
		return errors.New("cannot randomize Pointcheval-Sanders signature: please initialize curve properly")
	}
	rand, err := v.Curve.Rand()
	if err != nil {
		return errors.Wrap(err, "cannot randomize Pointcheval-Sanders signature")
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

// Serialize marshals Pointcheval-Sanders Signer
func (s *Signer) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

// Deserialize un-marshals Pointcheval-Sanders Signer
func (s *Signer) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, s)
}
