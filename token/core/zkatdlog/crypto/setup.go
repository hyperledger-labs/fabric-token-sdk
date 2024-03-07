/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/sha256"
	"encoding/json"
	"math"
	"strconv"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

const (
	DLogPublicParameters = "zkatdlog"
	DefaultPrecision     = uint64(64)
)

type RangeProofParams struct {
	LeftGenerators  []*mathlib.G1
	RightGenerators []*mathlib.G1
	P               *mathlib.G1
	Q               *mathlib.G1
	BitLength       int
	NumberOfRounds  int
}

func (rpp *RangeProofParams) Validate() error {
	if rpp.BitLength <= 0 {
		return errors.New("invalid range proof parameters: bit length is zero")
	}
	if rpp.NumberOfRounds <= 0 {
		return errors.New("invalid range proof parameters: number of rounds is zero")
	}
	if rpp.BitLength != int(math.Pow(2, float64(rpp.NumberOfRounds))) {
		return errors.Errorf("invalid range proof parameters: bit length should be %d\n", int(math.Pow(2, float64(rpp.NumberOfRounds))))
	}
	if len(rpp.LeftGenerators) != len(rpp.RightGenerators) {
		return errors.Errorf("invalid range proof parameters: the size of the left generators does not match the size of the right generators [%d vs, %d]", len(rpp.LeftGenerators), len(rpp.RightGenerators))
	}
	if len(rpp.LeftGenerators) != rpp.BitLength {
		return errors.Errorf("invalid range proof parameters: the size of the generators does not match the provided bit length [%d vs %d]", len(rpp.LeftGenerators), rpp.BitLength)
	}
	if rpp.Q == nil {
		return errors.New("invalid range proof parameters: generator Q is nil")
	}
	if rpp.P == nil {
		return errors.New("invalid range proof parameters: generator P is nil")
	}

	for i := 0; i < len(rpp.LeftGenerators); i++ {
		if rpp.LeftGenerators[i] == nil {
			return errors.Errorf("invalid range proof parameters: left generator at index %d is nil", i)
		}
		if rpp.RightGenerators[i] == nil {
			return errors.Errorf("invalid range proof parameters: right generator at index %d is nil", i)
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

type PublicParams struct {
	// Label is the label associated with the PublicParams.
	// It can be used by the driver for versioning purpose.
	Label string
	// Curve is the pairing-friendly elliptic curve used for everything but Idemix.
	Curve mathlib.CurveID
	// PedGen is the generator of the Pedersen commitment group.
	PedGen *mathlib.G1
	// PedParams contains the public parameters for the Pedersen commitment scheme.
	PedParams []*mathlib.G1
	// RangeProofParams contains the public parameters for the range proof scheme.
	RangeProofParams *RangeProofParams
	// IdemixCurveID is the pairing-friendly curve used for the idemix scheme.
	IdemixCurveID mathlib.CurveID
	// IdemixIssuerPK is the public key of the issuer of the idemix scheme.
	IdemixIssuerPK []byte
	// Auditor is the public key of the auditor.
	Auditor []byte
	// Issuers is a list of public keys of the entities that can issue tokens.
	Issuers [][]byte
	// MaxToken is the maximum quantity a token can hold
	MaxToken uint64
	// QuantityPrecision is the precision used to represent quantities
	QuantityPrecision uint64
}

func Setup(bitLength int, idemixIssuerPK []byte, idemixCurveID mathlib.CurveID) (*PublicParams, error) {
	return SetupWithCustomLabel(bitLength, idemixIssuerPK, DLogPublicParameters, idemixCurveID)
}

func SetupWithCustomLabel(bitLength int, idemixIssuerPK []byte, label string, idemixCurveID mathlib.CurveID) (*PublicParams, error) {
	pp := &PublicParams{Curve: mathlib.BN254}
	pp.Label = label
	if err := pp.GeneratePedersenParameters(); err != nil {
		return nil, errors.Wrapf(err, "failed to generated pedersen parameters")
	}
	if err := pp.GenerateRangeProofParameters(bitLength); err != nil {
		return nil, errors.Wrapf(err, "failed to generated range-proof parameters")
	}
	pp.IdemixIssuerPK = idemixIssuerPK
	pp.IdemixCurveID = idemixCurveID
	pp.RangeProofParams.BitLength = bitLength
	pp.RangeProofParams.NumberOfRounds = int(math.Log2(float64(bitLength)))
	pp.QuantityPrecision = DefaultPrecision
	pp.MaxToken = pp.ComputeMaxTokenValue()
	if err := pp.Validate(); err != nil {
		return nil, errors.Wrapf(err, "verification failed, invalid parameters")
	}

	return pp, nil
}

func (pp *PublicParams) IdemixCurve() mathlib.CurveID {
	return pp.IdemixCurveID
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
	return pp.MaxToken
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
	if err := json.Unmarshal(publicParams.Raw, pp); err != nil {
		return errors.Wrapf(err, "failed unmarshalling public parameters")
	}
	return nil
}

func (pp *PublicParams) GeneratePedersenParameters() error {
	curve := mathlib.Curves[pp.Curve]
	rand, err := curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	pp.PedGen = curve.GenG1.Mul(curve.NewRandomZr(rand))
	pp.PedParams = make([]*mathlib.G1, 3)

	for i := 0; i < len(pp.PedParams); i++ {
		pp.PedParams[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return nil
}

func (pp *PublicParams) GenerateRangeProofParameters(bitLength int) error {
	if bitLength <= 0 {
		return errors.Errorf("invalid bit length, must be larger than 0")
	}
	curve := mathlib.Curves[pp.Curve]

	pp.RangeProofParams = &RangeProofParams{
		P:              curve.HashToG1([]byte(strconv.Itoa(0))),
		Q:              curve.HashToG1([]byte(strconv.Itoa(1))),
		BitLength:      bitLength,
		NumberOfRounds: int(math.Log2(float64(bitLength))),
	}
	pp.RangeProofParams.LeftGenerators = make([]*mathlib.G1, bitLength)
	pp.RangeProofParams.RightGenerators = make([]*mathlib.G1, bitLength)

	for i := 0; i < bitLength; i++ {
		pp.RangeProofParams.LeftGenerators[i] = curve.HashToG1([]byte("RangeProof." + strconv.Itoa(2*(i+1))))
		pp.RangeProofParams.RightGenerators[i] = curve.HashToG1([]byte("RangeProof." + strconv.Itoa(2*(i+1)+1)))
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

func (pp *PublicParams) ComputeMaxTokenValue() uint64 {
	return uint64(math.Pow(2, float64(pp.RangeProofParams.BitLength))) - 1
}

func (pp *PublicParams) String() string {
	res, err := json.MarshalIndent(pp, " ", "  ")
	if err != nil {
		return err.Error()
	}
	return string(res)
}

func (pp *PublicParams) Validate() error {
	if int(pp.Curve) > len(mathlib.Curves)-1 {
		return errors.Errorf("invalid public parameters: invalid curveID [%d > %d]", int(pp.Curve), len(mathlib.Curves)-1)
	}
	if int(pp.IdemixCurveID) > len(mathlib.Curves)-1 {
		return errors.Errorf("invalid public parameters: invalid idemix curveID [%d > %d]", int(pp.Curve), len(mathlib.Curves)-1)
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
	maxToken := pp.ComputeMaxTokenValue()
	if maxToken != pp.MaxToken {
		return errors.Errorf("invalid maxt token, [%d]!=[%d]", maxToken, pp.MaxToken)
	}
	//if len(pp.Issuers) == 0 {
	//	return errors.New("invalid public parameters: empty list of issuers")
	//}
	return nil
}
