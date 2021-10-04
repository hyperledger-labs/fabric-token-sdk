/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/o2omp"
	"github.com/pkg/errors"
)

type Authorization struct {
	Type  *math.G1 // commitment to issuer's secret key and type (g_0^SK*g_1^type*h^r')
	Token *math.G1 // commitment to type and Value of a token (g_0^type*g_1^Value*h^r'')
	// (this corresponds to one of the issued tokens)
}

func NewAuthorization(typeNym, token *math.G1) *Authorization {
	return &Authorization{
		Type:  typeNym,
		Token: token,
	}
}

type AuthorizationWitness struct {
	Sk      *math.Zr // issuer's secret key
	TType   *math.Zr // type the issuer is authorized to issue
	TNymBF  *math.Zr // randomness in Type
	Value   *math.Zr // Value in token
	TokenBF *math.Zr // randomness in token
	Index   int      // index of Type
}

func NewWitness(sk, ttype, value, tNymBF, tokenBF *math.Zr, index int) *AuthorizationWitness {
	return &AuthorizationWitness{
		Sk:      sk,
		TType:   ttype,
		TNymBF:  tNymBF,
		Value:   value,
		TokenBF: tokenBF,
		Index:   index,
	}
}

type Signature struct {
	AuthorizationCorrectness []byte
	TypeCorrectness          []byte
}

func (s *Signature) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Signature) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, s)
}

type Signer struct {
	*Verifier
	Witness *AuthorizationWitness
}

// NewSigner initializes the prover
func NewSigner(witness *AuthorizationWitness, issuers []*math.G1, auth *Authorization, bitLength int, pp []*math.G1, c *math.Curve) *Signer {

	verifier := &Verifier{
		PedersenParams: pp,
		Issuers:        issuers,
		Auth:           auth,
		BitLength:      bitLength,
		Curve:          c,
	}
	return &Signer{
		Witness:  witness,
		Verifier: verifier,
	}
}

// check that the issuer knows the secret key of one of the commitments that link issuers to type
// check that type in the issued token is the same as type in the commitment NYM
func (s *Signer) Sign(message []byte) ([]byte, error) {
	if len(s.PedersenParams) != 3 {
		return nil, errors.Errorf("length of Pedersen parameters != 3")
	}

	// one out of many proofs
	commitments := make([]*math.G1, len(s.Issuers))
	for k, i := range s.Issuers {
		commitments[k] = s.Curve.NewG1()
		commitments[k] = s.Auth.Type.Copy()
		commitments[k].Sub(i)
	}
	o2omp := o2omp.NewProver(commitments, message, []*math.G1{s.PedersenParams[0], s.PedersenParams[2]}, s.BitLength, s.Witness.Index, s.Witness.TNymBF, s.Curve)

	sig := &Signature{}
	var err error
	sig.AuthorizationCorrectness, err = o2omp.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute issuer's signature")
	}

	w := NewTypeCorrectnessWitness(s.Witness.Sk, s.Witness.TType, s.Witness.Value, s.Witness.TNymBF, s.Witness.TokenBF)

	tcp := NewTypeCorrectnessProver(w, s.Auth.Type, s.Auth.Token, message, s.PedersenParams, s.Curve)
	sig.TypeCorrectness, err = tcp.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute issuer's signature")
	}

	return sig.Serialize()
}

func (s *Signer) Serialize() ([]byte, error) {
	v := &Verifier{Auth: s.Auth, Issuers: s.Issuers, PedersenParams: s.PedersenParams, BitLength: s.BitLength}
	return v.Serialize()
}

func (s *Signer) ToUniqueIdentifier() ([]byte, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ToUniqueIdentifier [%s]", string(raw))
	return []byte(hash.Hashable(raw).String()), nil
}

type Verifier struct {
	PedersenParams []*math.G1
	Issuers        []*math.G1 // g_0^skg_1^type
	Auth           *Authorization
	BitLength      int
	Curve          *math.Curve
}

func (v *Verifier) Verify(message, rawsig []byte) error {
	if len(v.PedersenParams) != 3 {
		return errors.Errorf("length of Pedersen parameters != 3")
	}

	sig := &Signature{}
	err := sig.Deserialize(rawsig)
	if err != nil {
		return errors.Errorf("failed to unmarshal issuer's signature")
	}
	commitments := make([]*math.G1, len(v.Issuers))
	for k, i := range v.Issuers {
		commitments[k] = v.Curve.NewG1()
		commitments[k] = v.Auth.Type.Copy()
		commitments[k].Sub(i)
	}

	// verify one out of many proof: issuer authorization
	err = o2omp.NewVerifier(commitments, message, []*math.G1{v.PedersenParams[0], v.PedersenParams[2]}, v.BitLength, v.Curve).Verify(sig.AuthorizationCorrectness)
	if err != nil {
		return errors.Wrapf(err, "failed to verify issuer's pseudonym")
	}

	// verify that type in authorization corresponds to type in token
	return NewTypeCorrectnessVerifier(v.Auth.Type, v.Auth.Token, message, v.PedersenParams, v.Curve).Verify(sig.TypeCorrectness)
}

func (v *Verifier) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

func (v *Verifier) Deserialize(bitLength int, issuers, pp []*math.G1, token *math.G1, raw []byte, curve *math.Curve) error {
	err := json.Unmarshal(raw, &v)
	if err != nil {
		return err
	}

	v.Auth.Token = token
	v.BitLength = bitLength
	v.PedersenParams = pp
	v.Issuers = issuers
	v.Curve = curve
	return nil
}
