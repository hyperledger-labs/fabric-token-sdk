/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package crypto

import (
	"encoding/json"
	math2 "math"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	DLogPublicParameters = "zkatdlog"
)

type PublicParams struct {
	P                *bn256.G1
	ZKATPedParams    []*bn256.G1
	RangeProofParams *RangeProofParams
	IdemixPK         []byte
	IssuingPolicy    []byte
	Auditor          []byte
	Label            string
}

type RangeProofParams struct {
	SignPK       []*bn256.G2
	SignedValues []*pssign.Signature
	Q            *bn256.G2
	Exponent     int
}

func NewPublicParamsFromBytes(raw []byte, label string) (*PublicParams, error) {
	pp := &PublicParams{}
	pp.Label = label
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed parsing public parameters")
	}
	return pp, nil
}

func (pp *PublicParams) Identifier() string {
	return pp.Label
}

func (pp *PublicParams) CertificationDriver() string {
	return pp.Label
}

func (pp *PublicParams) TokenDataHiding() bool {
	return true
}

func (pp *PublicParams) GraphHiding() bool {
	return false
}

func (pp *PublicParams) MaxTokenValue() uint64 {
	return uint64(len(pp.RangeProofParams.SignedValues)) - 1
}

func (pp *PublicParams) Bytes() ([]byte, error) {
	return pp.Serialize()
}

func (pp *PublicParams) Serialize() ([]byte, error) {
	raw, err := json.Marshal(pp)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&driver.SerializedPublicParameters{
		Identifier: pp.Label,
		Raw:        raw,
	})
}

func (pp *PublicParams) Deserialize(raw []byte) error {
	publicParams := &driver.SerializedPublicParameters{}
	if err := json.Unmarshal(raw, publicParams); err != nil {
		return err
	}
	if publicParams.Identifier != pp.Label {
		return errors.Errorf("invalid identifier, expecting [%s], got [%s]", pp.Label, publicParams.Identifier)
	}
	// logger.Debugf("unmarshall zkatdlog public params [%s]", string(publicParams.Raw))
	return json.Unmarshal(publicParams.Raw, pp)
}

func (pp *PublicParams) GeneratePedersenParameters() error {
	rand, err := bn256.GetRand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	pp.P = bn256.G1Gen().Mul(bn256.RandModOrder(rand))
	pp.ZKATPedParams = make([]*bn256.G1, 3)

	for i := 0; i < len(pp.ZKATPedParams); i++ {
		pp.ZKATPedParams[i] = bn256.G1Gen().Mul(bn256.RandModOrder(rand))
	}
	return nil
}

func (pp *PublicParams) GenerateRangeProofParameters(signer *pssign.Signer, maxValue int64) error {
	pp.RangeProofParams = &RangeProofParams{Q: signer.Q, SignPK: signer.PK}

	pp.RangeProofParams.SignedValues = make([]*pssign.Signature, maxValue)
	for i := 0; i < len(pp.RangeProofParams.SignedValues); i++ {
		var err error
		m := make([]*bn256.Zr, 1)
		m[0] = bn256.NewZrInt(i)
		pp.RangeProofParams.SignedValues[i], err = signer.Sign(m)
		if err != nil {
			return errors.Errorf("failed to generate public parameters: cannot sign range")
		}
	}

	return nil
}

func (pp *PublicParams) SetIssuingPolicy(issuers []*bn256.G1) error {
	ip := &IssuingPolicy{BitLength: int(math2.Ceil(math2.Log2(float64(len(issuers))))), IssuersNumber: len(issuers)}

	// pad list of issuers with a dummy commitment
	if len(issuers) != int(math2.Exp2(math2.Ceil(math2.Log2(float64(len(issuers)))))) {

		for i := len(issuers); i < int(math2.Exp2(math2.Ceil(math2.Log2(float64(len(issuers)))))); i++ {
			issuers = append(issuers, bn256.G1Gen())
		}
	}
	ip.Issuers = issuers
	var err error
	pp.IssuingPolicy, err = ip.Serialize()
	if err != nil {
		return err
	}

	return nil
}

func (pp *PublicParams) AddIssuer(issuer *bn256.G1) error {
	ip := &IssuingPolicy{}

	err := ip.Deserialize(pp.IssuingPolicy)
	if err != nil {
		return errors.Wrapf(err, "failed deserializing issuing policy")
	}

	ip.Issuers = append(ip.Issuers, issuer)
	if err := pp.SetIssuingPolicy(ip.Issuers); err != nil {
		return errors.Wrapf(err, "failed setting issuing policy")
	}
	return nil
}

func (pp *PublicParams) GetIssuingPolicy() (*IssuingPolicy, error) {
	ip := &IssuingPolicy{}
	err := ip.Deserialize(pp.IssuingPolicy)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing issuing policy")
	}
	return ip, nil
}

func Setup(base int64, exponent int, nymPK []byte) (*PublicParams, error) {
	return SetupWithCustomLabel(base, exponent, nymPK, DLogPublicParameters)
}

func SetupWithCustomLabel(base int64, exponent int, nymPK []byte, label string) (*PublicParams, error) {
	signer := &pssign.Signer{}
	err := signer.KeyGen(1)
	if err != nil {
		return nil, err
	}
	pp := &PublicParams{}
	pp.Label = label
	err = pp.GeneratePedersenParameters()
	if err != nil {
		return nil, err
	}
	err = pp.GenerateRangeProofParameters(signer, base)
	if err != nil {
		return nil, err
	}
	// empty issuing policy
	ip := &IssuingPolicy{}
	pp.IssuingPolicy, err = ip.Serialize()
	if err != nil {
		return nil, err
	}
	pp.IdemixPK = nymPK
	pp.RangeProofParams.Exponent = exponent
	// max value of any given token is max = base^exponent - 1
	return pp, nil
}
