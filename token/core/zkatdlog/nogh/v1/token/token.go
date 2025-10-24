/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/utils"
	noghv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Token encodes Type, Value, Owner
type Token comm.Token

func (t *Token) GetOwner() []byte {
	return t.Owner
}

// IsRedeem returns true if the token has an empty owner field
func (t *Token) IsRedeem() bool {
	return len(t.Owner) == 0
}

// Serialize marshals Token
func (t *Token) Serialize() ([]byte, error) {
	data, err := utils.ToProtoG1(t.Data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize output")
	}
	raw, err := proto.Marshal(&actions.Token{
		Owner: t.Owner,
		Data:  data,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed serializing token")
	}
	return comm.WrapTokenWithType(raw)
}

// Deserialize unmarshals Token
func (t *Token) Deserialize(bytes []byte) error {
	typed, err := comm.UnmarshalTypedToken(bytes)
	if err != nil {
		return errors.Wrapf(err, "failed deserializing token")
	}
	token := &actions.Token{}
	if err := proto.Unmarshal(typed.Token, token); err != nil {
		return errors.Wrapf(err, "failed unmarshalling token")
	}
	t.Owner = token.Owner
	t.Data, err = utils.FromG1Proto(token.Data)
	return err
}

// ToClear returns Token in the clear
func (t *Token) ToClear(meta *Metadata, pp *noghv1.PublicParams) (*token.Token, error) {
	com, err := Commit([]*math.Zr{math.Curves[pp.Curve].HashToZr([]byte(meta.Type)), meta.Value, meta.BlindingFactor}, pp.PedersenGenerators, math.Curves[pp.Curve])
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve token in the clear: failed to check token data")
	}
	// check that token matches meta
	if !com.Equals(t.Data) {
		return nil, errors.New("cannot retrieve token in the clear: output does not match provided opening")
	}
	return &token.Token{
		Type:     meta.Type,
		Quantity: "0x" + meta.Value.String(),
		Owner:    t.Owner,
	}, nil
}

func (t *Token) Validate(checkOwner bool) error {
	if checkOwner && len(t.Owner) == 0 {
		return errors.Errorf("token owner cannot be empty")
	}
	if t.Data == nil {
		return errors.Errorf("token data cannot be empty")
	}
	return nil
}

func computeTokens(tw []*Metadata, pp []*math.G1, c *math.Curve) ([]*math.G1, error) {
	tokens := make([]*math.G1, len(tw))
	var err error
	for i := range tw {
		hash := c.HashToZr([]byte(tw[i].Type))
		tokens[i], err = Commit([]*math.Zr{hash, tw[i].Value, tw[i].BlindingFactor}, pp, c)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to compute token [%d]", i)
		}
	}

	return tokens, nil
}

func GetTokensWithWitness(values []uint64, tokenType token.Type, pp []*math.G1, c *math.Curve) ([]*math.G1, []*Metadata, error) {
	if c == nil {
		return nil, nil, errors.New("cannot get tokens with witness: please initialize curve")
	}
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot get tokens with witness")
	}
	tw := make([]*Metadata, len(values))
	for i, v := range values {
		tw[i] = &Metadata{
			BlindingFactor: c.NewRandomZr(rand),
			Value:          c.NewZrFromUint64(v),
			Type:           tokenType,
		}
	}
	tokens, err := computeTokens(tw, pp, c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot get tokens with witness")
	}
	return tokens, tw, nil
}

// Metadata contains the metadata of a token
type Metadata comm.Metadata

// NewMetadata returns an array of Metadata that corresponds to the passed arguments
func NewMetadata(curve math.CurveID, tokenType token.Type, values []uint64, bfs []*math.Zr) []*Metadata {
	witness := make([]*Metadata, len(values))
	for i, v := range values {
		witness[i] = &Metadata{Value: math.Curves[curve].NewZrFromUint64(v), BlindingFactor: bfs[i]}
	}
	witness[0].Type = tokenType
	return witness
}

// Deserialize un-marshals Metadata
func (m *Metadata) Deserialize(b []byte) error {
	typed, err := comm.UnmarshalTypedMetadata(b)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize metadata")
	}
	metadata := &actions.TokenMetadata{}
	if err := proto.Unmarshal(typed.Metadata, metadata); err != nil {
		return errors.Wrapf(err, "failed to deserialize metadata")
	}
	m.Type = token.Type(metadata.Type)
	m.Value, err = utils.FromZrProto(metadata.Value)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize metadata")
	}
	m.BlindingFactor, err = utils.FromZrProto(metadata.BlindingFactor)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize metadata")
	}
	if metadata.Issuer != nil {
		m.Issuer = metadata.Issuer.Raw
	}
	return nil
}

// Serialize un-marshals Metadata
func (m *Metadata) Serialize() ([]byte, error) {
	value, err := utils.ToProtoZr(m.Value)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize metadata")
	}
	blindingFactor, err := utils.ToProtoZr(m.BlindingFactor)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize metadata")
	}
	raw, err := proto.Marshal(&actions.TokenMetadata{
		Type:           string(m.Type),
		Value:          value,
		BlindingFactor: blindingFactor,
		Issuer:         &pp.Identity{Raw: m.Issuer},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize metadata")
	}
	return comm.WrapMetadataWithType(raw)
}

func (m *Metadata) Clone() *Metadata {
	return &Metadata{
		Type:           m.Type,
		BlindingFactor: m.BlindingFactor,
		Issuer:         m.Issuer,
		Value:          m.Value,
	}
}

// Commit computes the Pedersen commitment of the passed elements using the passed bases
func Commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) (*math.G1, error) {
	if len(generators) != len(vector) {
		return nil, errors.Errorf("number of generators is not equal to number of vector elements, [%d]!=[%d]", len(generators), len(vector))
	}
	com := c.NewG1()
	for i := range vector {
		if vector[i] == nil {
			return nil, errors.New("cannot commit a nil element")
		}
		com.Add(generators[i].Mul(vector[i]))
	}
	return com, nil
}

type UpgradeWitness struct {
	FabToken *fabtokenv1.Output
	// BlindingFactor is the blinding factor used to commit type and value
	BlindingFactor *math.Zr
}

func (u *UpgradeWitness) Validate() error {
	if u.FabToken == nil {
		return errors.New("missing FabToken")
	}
	if len(u.FabToken.Owner) == 0 {
		return errors.New("missing FabToken.Owner")
	}
	if len(u.FabToken.Type) == 0 {
		return errors.New("missing FabToken.Type")
	}
	if len(u.FabToken.Quantity) == 0 {
		return errors.New("missing FabToken.Quantity")
	}
	if u.BlindingFactor == nil {
		return errors.New("missing BlindingFactor")
	}
	return nil
}
