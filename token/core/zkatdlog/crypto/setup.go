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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

const (
	DLogPublicParameters = "zkatdlog"
	DefaultPrecision     = uint64(64)
)

type PublicParams struct {
	// Label is the label associated with the PublicParams.
	// It can be used by the driver for versioning purpose.
	Label string
	// Curve is the pairing-friendly elliptic curve used for everything but Idemix.
	Curve math.CurveID
	// PedGen is the generator of the Pedersen commitment group.
	PedGen *math.G1
	// PedParams contains the public parameters for the Pedersen commitment scheme.
	PedParams []*math.G1
	// RangeProofParams contains the public parameters for the range proof scheme.
	RangeProofParams *RangeProofParams
	// IdemixCurveID is the pairing-friendly curve used for the idemix scheme.
	IdemixCurveID math.CurveID
	// IdemixIssuerPK is the public key of the issuer of the idemix scheme.
	IdemixIssuerPK []byte
	// Auditor is the public key of the auditor.
	Auditor []byte
	// Issuers is a list of public keys of the entities that can issue tokens.
	Issuers [][]byte
	// QuantityPrecision is the precision used to represent quantities
	QuantityPrecision uint64
}

type RangeProofParams struct {
	SignPK       []*math.G2
	SignedValues []*pssign.Signature
	Q            *math.G2
	Exponent     int
}

func (rpp *RangeProofParams) Validate() error {
	if len(rpp.SignPK) != 3 {
		return errors.Errorf("invalid range proof parameters: signature public key should be 3, instead it is %d", len(rpp.SignPK))
	}
	if len(rpp.SignedValues) < 2 {
		return errors.New("invalid range proof parameters: signed values should be > 2")
	}
	if rpp.Q == nil {
		return errors.New("invalid range proof parameters: generator Q is nil")
	}
	if rpp.Exponent == 0 {
		return errors.New("invalid range proof parameters: exponent is 0")
	}
	for i := 0; i < len(rpp.SignedValues); i++ {
		if rpp.SignedValues[i] == nil {
			return errors.Errorf("invalid range proof parameters: signed value at index %d is nil", i)
		}
	}
	for i := 0; i < len(rpp.SignPK); i++ {
		if rpp.SignPK[i] == nil {
			return errors.Errorf("invalid range proof parameters: public key at index %d is nil", i)
		}
	}
	return nil
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

func (pp *PublicParams) Auditors() []view.Identity {
	return []view.Identity{pp.Auditor}
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

func (pp *PublicParams) Precision() uint64 {
	return pp.QuantityPrecision
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
	// TODO: perform additional checks:
	// the curve exists
	// the idemix params are all set,
	// and so on
	return nil
}

func (pp *PublicParams) GeneratePedersenParameters() error {
	curve := math.Curves[pp.Curve]
	rand, err := curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	pp.PedGen = curve.GenG1.Mul(curve.NewRandomZr(rand))
	pp.PedParams = make([]*math.G1, 3)

	for i := 0; i < len(pp.PedParams); i++ {
		pp.PedParams[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
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

func (pp *PublicParams) AddAuditor(auditor view.Identity) {
	pp.Auditor = auditor
}

func (pp *PublicParams) AddIssuer(id view.Identity) {
	pp.Issuers = append(pp.Issuers, id)
}

func (pp *PublicParams) ComputeHash() ([]byte, error) {
	raw, err := pp.Bytes()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to serialize public params")
	}
	hash := sha256.New()
	n, err := hash.Write(raw)
	if n != len(raw) {
		return nil, errors.New("failed to hash public parameters")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash public parameters")
	}
	return hash.Sum(nil), nil
}

func Setup(base int64, exponent int, nymPK []byte, idemixCurveID math.CurveID) (*PublicParams, error) {
	return SetupWithCustomLabel(base, exponent, nymPK, DLogPublicParameters, idemixCurveID)
}

func SetupWithCustomLabel(base int64, exponent int, nymPK []byte, label string, idemixCurveID math.CurveID) (*PublicParams, error) {
	signer := pssign.NewSigner(nil, nil, nil, math.Curves[math.BN254])
	err := signer.KeyGen(1)
	if err != nil {
		return nil, err
	}
	pp := &PublicParams{Curve: math.BN254}
	pp.Label = label
	err = pp.GeneratePedersenParameters()
	if err != nil {
		return nil, err
	}
	err = pp.GenerateRangeProofParameters(signer, base)
	if err != nil {
		return nil, err
	}
	pp.IdemixIssuerPK = nymPK
	pp.IdemixCurveID = idemixCurveID
	pp.RangeProofParams.Exponent = exponent
	pp.QuantityPrecision = DefaultPrecision
	// max value of any given token is max = base^exponent - 1
	return pp, nil
}

func (pp *PublicParams) Validate() error {
	if int(pp.Curve) > len(math.Curves)-1 {
		return errors.Errorf("invalid public parameters: invalid curveID [%d > %d]", int(pp.Curve), len(math.Curves)-1)
	}
	if int(pp.IdemixCurveID) > len(math.Curves)-1 {
		return errors.Errorf("invalid public parameters: invalid idemix curveID [%d > %d]", int(pp.Curve), len(math.Curves)-1)
	}
	if pp.PedGen == nil {
		return errors.New("invalid public parameters: nil Pedersen generator")
	}
	if len(pp.PedParams) != 3 {
		return errors.Errorf("invalid public parameters: length mismatch in Pedersen parameters [%d vs. 3]", len(pp.PedParams))
	}
	for i := 0; i < len(pp.PedParams); i++ {
		if pp.PedParams[i] == nil {
			return errors.Errorf("invalid public parameters: nil Pedersen parameter at index %d", i)
		}
	}
	if pp.RangeProofParams == nil {
		return errors.New("invalid public parameters: nil range proof parameters")
	}
	err := pp.RangeProofParams.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid public parameters")
	}
	if pp.QuantityPrecision != DefaultPrecision {
		return errors.Errorf("invalid public parameters: quantity precision should be %d instead it is %d", DefaultPrecision, pp.QuantityPrecision)
	}
	if len(pp.IdemixIssuerPK) == 0 {
		return errors.New("invalid public parameters: empty idemix issuer")
	}
	//if len(pp.Issuers) == 0 {
	//	return errors.New("invalid public parameters: empty list of issuers")
	//}
	return nil
}
