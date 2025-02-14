/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"crypto/sha256"
	"math/bits"
	"strconv"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
	"github.com/pkg/errors"
)

const (
	DLogPublicParameters = "zkatdlog"
	Version              = "1.0.0"
)

var (
	SupportedPrecisions = []uint64{16, 32, 64}
)

type RangeProofParams struct {
	LeftGenerators  []*mathlib.G1
	RightGenerators []*mathlib.G1
	P               *mathlib.G1
	Q               *mathlib.G1
	BitLength       uint64
	NumberOfRounds  uint64
}

func (rpp *RangeProofParams) Validate(curveID mathlib.CurveID) error {
	if rpp.BitLength == 0 {
		return errors.New("invalid range proof parameters: bit length is zero")
	}
	if rpp.NumberOfRounds == 0 {
		return errors.New("invalid range proof parameters: number of rounds is zero")
	}
	if rpp.NumberOfRounds > 64 {
		return errors.New("invalid range proof parameters: number of rounds must be smaller or equal to 64")
	}
	if rpp.BitLength != uint64(1<<rpp.NumberOfRounds) {
		return errors.Errorf("invalid range proof parameters: bit length should be %d\n", uint64(1<<rpp.NumberOfRounds))
	}
	if len(rpp.LeftGenerators) != len(rpp.RightGenerators) {
		return errors.Errorf("invalid range proof parameters: the size of the left generators does not match the size of the right generators [%d vs, %d]", len(rpp.LeftGenerators), len(rpp.RightGenerators))
	}
	if err := math.CheckElement(rpp.Q, curveID); err != nil {
		return errors.Wrapf(err, "invalid range proof parameters: generator Q is invalid")
	}
	if err := math.CheckElement(rpp.P, curveID); err != nil {
		return errors.Wrapf(err, "invalid range proof parameters: generator P is invalid")
	}
	if err := math.CheckElements(rpp.LeftGenerators, curveID, rpp.BitLength); err != nil {
		return errors.Wrap(err, "invalid range proof parameters, left generators is invalid")
	}
	if err := math.CheckElements(rpp.RightGenerators, curveID, rpp.BitLength); err != nil {
		return errors.Wrap(err, "invalid range proof parameters, right generators is invalid")
	}

	return nil
}

type IdemixIssuerPublicKey struct {
	PublicKey []byte
	Curve     mathlib.CurveID
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
	// Ver is the version of these public params
	Ver string
	// Curve is the pairing-friendly elliptic curve used for everything but Idemix.
	Curve mathlib.CurveID
	// PedersenGenerators contains the public parameters for the Pedersen commitment scheme.
	PedersenGenerators []*mathlib.G1
	// RangeProofParams contains the public parameters for the range proof scheme.
	RangeProofParams *RangeProofParams
	// IdemixIssuerPublicKeys contains the idemix issuer public keys
	// Wallets should prefer the use of keys valid under the public key whose index in the array is the smallest.
	IdemixIssuerPublicKeys []IdemixIssuerPublicKey
	// Auditor is the public key of the auditor.
	Auditor driver.Identity
	// IssuerIDs is a list of public keys of the entities that can issue tokens.
	IssuerIDs []driver.Identity
	// MaxToken is the maximum quantity a token can hold
	MaxToken uint64
	// QuantityPrecision is the precision used to represent quantities
	QuantityPrecision uint64
}

func Setup(bitLength uint64, idemixIssuerPK []byte, idemixCurveID mathlib.CurveID) (*PublicParams, error) {
	return setup(bitLength, idemixIssuerPK, DLogPublicParameters, idemixCurveID)
}

func setup(bitLength uint64, idemixIssuerPK []byte, label string, idemixCurveID mathlib.CurveID) (*PublicParams, error) {
	if bitLength > 64 {
		return nil, errors.Errorf("invalid bit length [%d], should be smaller than 64", bitLength)
	}
	if bitLength == 0 {
		return nil, errors.New("invalid bit length, should be greater than 0")
	}
	pp := &PublicParams{
		Label: label,
		Curve: mathlib.BN254,
		Ver:   Version,
		IdemixIssuerPublicKeys: []IdemixIssuerPublicKey{
			{
				PublicKey: idemixIssuerPK,
				Curve:     idemixCurveID,
			},
		},
		QuantityPrecision: bitLength,
	}
	if err := pp.GeneratePedersenParameters(); err != nil {
		return nil, errors.Wrapf(err, "failed to generated pedersen parameters")
	}
	if err := pp.GenerateRangeProofParameters(bitLength); err != nil {
		return nil, errors.Wrapf(err, "failed to generated range-proof parameters")
	}
	pp.RangeProofParams.BitLength = bitLength
	pp.RangeProofParams.NumberOfRounds = log2(bitLength)
	pp.MaxToken = pp.ComputeMaxTokenValue()
	return pp, nil
}

func (pp *PublicParams) Identifier() string {
	return pp.Label
}

func (pp *PublicParams) Version() string {
	return pp.Ver
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

func (pp *PublicParams) Auditors() []driver.Identity {
	if len(pp.Auditor) == 0 {
		return []driver.Identity{}
	}
	return []driver.Identity{pp.Auditor}
}

// Issuers returns the list of authorized issuers
func (pp *PublicParams) Issuers() []driver.Identity {
	return pp.IssuerIDs
}

func (pp *PublicParams) Serialize() ([]byte, error) {
	raw, err := json.Marshal(pp)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&pp2.PublicParameters{
		Identifier: pp.Label,
		Raw:        raw,
	})
}

func (pp *PublicParams) Precision() uint64 {
	return pp.QuantityPrecision
}

func (pp *PublicParams) Deserialize(raw []byte) error {
	publicParams := &pp2.PublicParameters{}
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
	pp.PedersenGenerators = make([]*mathlib.G1, 3)

	for i := 0; i < len(pp.PedersenGenerators); i++ {
		pp.PedersenGenerators[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return nil
}

func (pp *PublicParams) GenerateRangeProofParameters(bitLength uint64) error {
	curve := mathlib.Curves[pp.Curve]

	pp.RangeProofParams = &RangeProofParams{
		P:              curve.HashToG1([]byte(strconv.Itoa(0))),
		Q:              curve.HashToG1([]byte(strconv.Itoa(1))),
		BitLength:      bitLength,
		NumberOfRounds: log2(bitLength),
	}
	pp.RangeProofParams.LeftGenerators = make([]*mathlib.G1, bitLength)
	pp.RangeProofParams.RightGenerators = make([]*mathlib.G1, bitLength)

	for i := uint64(0); i < bitLength; i++ {
		pp.RangeProofParams.LeftGenerators[i] = curve.HashToG1([]byte("RangeProof." + strconv.FormatUint(2*(i+1), 10)))
		pp.RangeProofParams.RightGenerators[i] = curve.HashToG1([]byte("RangeProof." + strconv.FormatUint(2*(i+1)+1, 10)))
	}

	return nil
}

func (pp *PublicParams) AddAuditor(auditor driver.Identity) {
	pp.Auditor = auditor
}

func (pp *PublicParams) AddIssuer(id driver.Identity) {
	pp.IssuerIDs = append(pp.IssuerIDs, id)
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
	return 1<<pp.RangeProofParams.BitLength - 1
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
	if len(pp.IdemixIssuerPublicKeys) != 1 {
		return errors.Errorf("expected one idemix issuer public key, found [%d]", len(pp.IdemixIssuerPublicKeys))
	}

	for _, issuer := range pp.IdemixIssuerPublicKeys {
		if len(issuer.PublicKey) == 0 {
			return errors.Errorf("expected idemix issuer public key to be non-empty")
		}
		if int(issuer.Curve) > len(mathlib.Curves)-1 {
			return errors.Errorf("invalid public parameters: invalid idemix curveID [%d > %d]", int(pp.Curve), len(mathlib.Curves)-1)
		}
	}
	if err := math.CheckElements(pp.PedersenGenerators, pp.Curve, 3); err != nil {
		return errors.Wrapf(err, "invalid pedersen generators")
	}
	if pp.RangeProofParams == nil {
		return errors.New("invalid public parameters: nil range proof parameters")
	}
	bitLength := pp.RangeProofParams.BitLength
	supportedPrecisions := collections.NewSet(SupportedPrecisions...)
	if !supportedPrecisions.Contains(bitLength) {
		return errors.Errorf("invalid bit length [%d], should be one of [%v]", bitLength, supportedPrecisions.ToSlice())
	}
	err := pp.RangeProofParams.Validate(pp.Curve)
	if err != nil {
		return errors.Wrap(err, "invalid public parameters")
	}
	if pp.QuantityPrecision != pp.RangeProofParams.BitLength {
		return errors.Errorf("invalid public parameters: quantity precision should be [%d] instead it is [%d]", pp.RangeProofParams.BitLength, pp.QuantityPrecision)
	}
	maxToken := pp.ComputeMaxTokenValue()
	if maxToken != pp.MaxToken {
		return errors.Errorf("invalid maxt token, [%d]!=[%d]", maxToken, pp.MaxToken)
	}
	// if len(pp.Issuers) == 0 {
	//	return errors.New("invalid public parameters: empty list of issuers")
	// }
	return nil
}

func log2(x uint64) uint64 {
	return 63 - uint64(bits.LeadingZeros64(x))
}
