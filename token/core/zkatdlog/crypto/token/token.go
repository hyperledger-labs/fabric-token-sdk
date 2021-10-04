/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"encoding/json"

	bn256 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Token struct {
	Owner []byte    // this could either be msp identity or an idemix identity
	Data  *bn256.G1 // Commitments to type and value
}

func (t *Token) IsRedeem() bool {
	return len(t.Owner) == 0
}

func (t *Token) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Token) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, t)
}

func (t *Token) GetCommitment() *bn256.G1 {
	return t.Data
}

func (t *Token) GetTokenInTheClear(inf *TokenInformation, pp *crypto.PublicParams) (*token2.Token, error) {
	com, err := common.ComputePedersenCommitment([]*bn256.Zr{bn256.Curves[pp.Curve].HashToZr([]byte(inf.Type)), inf.Value, inf.BlindingFactor}, pp.ZKATPedParams, bn256.Curves[pp.Curve])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check token data")
	}
	// check that token matches inf
	if !com.Equals(t.Data) {
		return nil, errors.Errorf("output does not math provided opening")
	}
	// todo identity mixer opening is missing
	return &token2.Token{
		Type:     inf.Type,
		Quantity: "0x" + inf.Value.String(),
		Owner:    &token2.Owner{Raw: t.Owner},
	}, nil
}

func computeTokens(tw []*TokenDataWitness, pp []*bn256.G1, c *bn256.Curve) ([]*bn256.G1, error) {
	tokens := make([]*bn256.G1, len(tw))
	var err error
	for i := 0; i < len(tw); i++ {
		typehash := c.HashToZr([]byte(tw[i].Type))
		tokens[i], err = common.ComputePedersenCommitment([]*bn256.Zr{typehash, tw[i].Value, tw[i].BlindingFactor}, pp, c)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to compute token")
		}
	}

	return tokens, nil
}

func GetTokensWithWitness(values []uint64, ttype string, pp []*bn256.G1, c *bn256.Curve) ([]*bn256.G1, []*TokenDataWitness, error) {
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, errors.Errorf("failed to get random number generator")
	}
	tw := make([]*TokenDataWitness, len(values))
	for i, v := range values {
		tw[i] = &TokenDataWitness{}
		tw[i].BlindingFactor = c.NewRandomZr(rand)
		tw[i].Value = c.NewZrFromInt(int64(v)) // todo .SetUint64(v)
		tw[i].Type = ttype
	}
	tokens, err := computeTokens(tw, pp, c)
	if err != nil {
		return nil, nil, err
	}
	return tokens, tw, nil
}

// information underlying token: this includes owner and token data witness
type TokenInformation struct {
	Type           string
	Value          *bn256.Zr
	BlindingFactor *bn256.Zr
	Owner          []byte
	Issuer         []byte
}

func (inf *TokenInformation) Deserialize(b []byte) error {
	return json.Unmarshal(b, inf)
}

func (inf *TokenInformation) Serialize() ([]byte, error) {
	return json.Marshal(inf)
}

// witness of token data
type TokenDataWitness struct {
	Type           string
	Value          *bn256.Zr
	BlindingFactor *bn256.Zr
}
