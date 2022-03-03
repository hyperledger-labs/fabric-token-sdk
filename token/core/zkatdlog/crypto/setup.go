/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/sha256"
	"encoding/json"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	DLogPublicParameters = "zkatdlog"
)

type PublicParams struct {
	P                *math.G1
	ZKATPedParams    []*math.G1
	RangeProofParams *RangeProofParams
	IdemixCurve      math.CurveID
	IdemixPK         []byte
	IssuingPolicy    []byte
	Auditor          []byte
	Issuers          [][]byte
	Label            string
	Curve            int

	Hash []byte
}

type RangeProofParams struct {
	SignPK       []*math.G2
	SignedValues []*pssign.Signature
	Q            *math.G2
	Exponent     int
}

func NewPublicParamsFromBytes(raw []byte, label string) (*PublicParams, error) {
	pp := &PublicParams{}
	pp.Label = label
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed parsing public parameters")
	}
	if err := pp.ComputeHash(raw); err != nil {
		return nil, errors.Wrap(err, "failed computing hash")
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
	if err := json.Unmarshal(publicParams.Raw, pp); err != nil {
		return errors.Wrapf(err, "failed unmarshalling public parameters")
	}
	if err := pp.ComputeHash(raw); err != nil {
		return errors.Wrap(err, "failed computing hash")
	}
	return nil
}

func (pp *PublicParams) GeneratePedersenParameters() error {
	curve := math.Curves[pp.Curve]
	rand, err := curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	pp.P = curve.GenG1.Mul(curve.NewRandomZr(rand))
	pp.ZKATPedParams = make([]*math.G1, 3)

	for i := 0; i < len(pp.ZKATPedParams); i++ {
		pp.ZKATPedParams[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return nil
}

func (pp *PublicParams) GenerateRangeProofParameters(signer *pssign.Signer, maxValue int64) error {
	curve := math.Curves[pp.Curve]
	pp.RangeProofParams = &RangeProofParams{Q: signer.Q, SignPK: signer.PK}

	pp.RangeProofParams.SignedValues = make([]*pssign.Signature, maxValue)
	for i := 0; i < len(pp.RangeProofParams.SignedValues); i++ {
		var err error
		m := make([]*math.Zr, 1)
		m[0] = curve.NewZrFromInt(int64(i))
		pp.RangeProofParams.SignedValues[i], err = signer.Sign(m)
		if err != nil {
			return errors.Errorf("failed to generate public parameters: cannot sign range")
		}
	}

	return nil
}

func (pp *PublicParams) ComputeHash(raw []byte) error {
	hash := sha256.New()
	n, err := hash.Write(raw)
	if n != len(raw) {
		return errors.New("failed to hash public parameters")
	}
	if err != nil {
		return errors.Wrap(err, "failed to hash public parameters")
	}
	pp.Hash = hash.Sum(nil)
	return nil
}

func (pp *PublicParams) AddAuditor(auditor view.Identity) {
	pp.Auditor = auditor
}

func (pp *PublicParams) AddIssuer(id view.Identity) {
	pp.Issuers = append(pp.Issuers, id)
}

func Setup(base int64, exponent int, nymPK []byte, curveID math.CurveID) (*PublicParams, error) {
	return SetupWithCustomLabel(base, exponent, nymPK, DLogPublicParameters, curveID)
}

func SetupWithCustomLabel(base int64, exponent int, nymPK []byte, label string, curveID math.CurveID) (*PublicParams, error) {
	signer := pssign.NewSigner(nil, nil, nil, math.Curves[1])
	err := signer.KeyGen(1)
	if err != nil {
		return nil, err
	}
	pp := &PublicParams{Curve: 1}
	pp.Label = label
	err = pp.GeneratePedersenParameters()
	if err != nil {
		return nil, err
	}
	err = pp.GenerateRangeProofParameters(signer, base)
	if err != nil {
		return nil, err
	}
	pp.IdemixPK = nymPK
	pp.IdemixCurve = curveID
	pp.RangeProofParams.Exponent = exponent
	// max value of any given token is max = base^exponent - 1
	return pp, nil
}
