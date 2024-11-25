/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// Token encodes Type, Value, Owner
type Token struct {
	// Owner is the owner of the token
	Owner []byte
	// Data is the Pedersen commitment to type and value
	Data *math.G1
}

func (t *Token) GetOwner() []byte {
	return t.Owner
}

// IsRedeem returns true if the token has an empty owner field
func (t *Token) IsRedeem() bool {
	return len(t.Owner) == 0
}

// Serialize marshals Token
func (t *Token) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Deserialize unmarshals Token
func (t *Token) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, t)
}

// GetCommitment returns the Pedersen commitment in Token
func (t *Token) GetCommitment() *math.G1 {
	return t.Data
}

// GetTokenInTheClear returns Token in the clear
func (t *Token) GetTokenInTheClear(meta *Metadata, pp *crypto.PublicParams) (*token2.Token, error) {
	com, err := commit([]*math.Zr{math.Curves[pp.Curve].HashToZr([]byte(meta.Type)), meta.Value, meta.BlindingFactor}, pp.PedersenGenerators, math.Curves[pp.Curve])
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve token in the clear: failed to check token data")
	}
	// check that token matches meta
	if !com.Equals(t.Data) {
		return nil, errors.New("cannot retrieve token in the clear: output does not match provided opening")
	}
	return &token2.Token{
		Type:     meta.Type,
		Quantity: "0x" + meta.Value.String(),
		Owner:    t.Owner,
	}, nil
}

func computeTokens(tw []*TokenDataWitness, pp []*math.G1, c *math.Curve) ([]*math.G1, error) {
	tokens := make([]*math.G1, len(tw))
	var err error
	for i := 0; i < len(tw); i++ {
		hash := c.HashToZr([]byte(tw[i].Type))
		tokens[i], err = commit([]*math.Zr{hash, c.NewZrFromInt(int64(tw[i].Value)), tw[i].BlindingFactor}, pp, c)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to compute token [%d]", i)
		}
	}

	return tokens, nil
}

func GetTokensWithWitness(values []uint64, ttype string, pp []*math.G1, c *math.Curve) ([]*math.G1, []*TokenDataWitness, error) {
	if c == nil {
		return nil, nil, errors.New("cannot get tokens with witness: please initialize curve")
	}
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot get tokens with witness")
	}
	tw := make([]*TokenDataWitness, len(values))
	for i, v := range values {
		tw[i] = &TokenDataWitness{
			BlindingFactor: c.NewRandomZr(rand),
			Value:          v,
			Type:           ttype,
		}
	}
	tokens, err := computeTokens(tw, pp, c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot get tokens with witness")
	}
	return tokens, tw, nil
}

// Metadata contains the metadata of a token
type Metadata struct {
	// Type is the type of the token
	Type string
	// Value is the quantity of the token
	Value *math.Zr
	// BlindingFactor is the blinding factor used to commit type and value
	BlindingFactor *math.Zr
	// Owner is the owner of the token
	Owner []byte
	// Issuer is the issuer of the token, if defined
	Issuer []byte
}

// Deserialize un-marshals Metadata
func (m *Metadata) Deserialize(b []byte) error {
	return json.Unmarshal(b, m)
}

// Serialize un-marshals Metadata
func (m *Metadata) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// TokenDataWitness contains the opening of Data in Token
type TokenDataWitness struct {
	Type           string
	Value          uint64
	BlindingFactor *math.Zr
}

// Clone produces a copy of TokenDataWitness
func (tdw *TokenDataWitness) Clone() *TokenDataWitness {
	return &TokenDataWitness{
		Type:           tdw.Type,
		Value:          tdw.Value,
		BlindingFactor: tdw.BlindingFactor.Copy(),
	}
}

// NewTokenDataWitness returns an array of TokenDataWitness that corresponds to the passed arguments
func NewTokenDataWitness(ttype string, values []uint64, bfs []*math.Zr) []*TokenDataWitness {
	witness := make([]*TokenDataWitness, len(values))
	for i, v := range values {
		witness[i] = &TokenDataWitness{Value: v, BlindingFactor: bfs[i]}
	}
	witness[0].Type = ttype
	return witness
}

func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) (*math.G1, error) {
	com := c.NewG1()
	for i := range vector {
		if vector[i] == nil {
			return nil, errors.New("cannot commit a nil element")
		}
		com.Add(generators[i].Mul(vector[i]))
	}
	return com, nil
}
