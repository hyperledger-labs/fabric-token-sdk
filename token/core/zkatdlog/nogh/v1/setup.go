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
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/pp"
	utils2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/protos"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
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

func (rpp *RangeProofParams) ToProtos() (*pp.RangeProofParams, error) {
	lefGenerators, err := utils2.ToProtoG1Slice(rpp.LeftGenerators)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert left generators to protos")
	}
	rightGenerators, err := utils2.ToProtoG1Slice(rpp.RightGenerators)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert right generators to protos")
	}
	p, err := utils2.ToProtoG1(rpp.P)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert p to proto")
	}
	q, err := utils2.ToProtoG1(rpp.Q)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert q to proto")
	}

	rangeProofParams := &pp.RangeProofParams{
		LeftGenerators:  lefGenerators,
		RightGenerators: rightGenerators,
		P:               p,
		Q:               q,
		BitLength:       rpp.BitLength,
		NumberOfRounds:  rpp.NumberOfRounds,
	}
	return rangeProofParams, nil
}

func (rpp *RangeProofParams) FromProto(params *pp.RangeProofParams) error {
	rpp.NumberOfRounds = params.NumberOfRounds
	rpp.BitLength = params.BitLength
	var err error
	rpp.LeftGenerators, err = utils2.FromG1ProtoSlice(params.LeftGenerators)
	if err != nil {
		return errors.Wrapf(err, "failed to convert left generators to protos")
	}
	rpp.RightGenerators, err = utils2.FromG1ProtoSlice(params.RightGenerators)
	if err != nil {
		return errors.Wrapf(err, "failed to convert right generators to protos")
	}
	rpp.P, err = utils2.FromG1Proto(params.P)
	if err != nil {
		return errors.Wrapf(err, "failed to convert p to proto")
	}
	rpp.Q, err = utils2.FromG1Proto(params.Q)
	if err != nil {
		return errors.Wrapf(err, "failed to convert q to proto")
	}
	return nil
}

type IdemixIssuerPublicKey struct {
	PublicKey []byte
	Curve     mathlib.CurveID
}

func (i *IdemixIssuerPublicKey) ToProtos() (*pp.IdemixIssuerPublicKey, error) {
	return &pp.IdemixIssuerPublicKey{
		PublicKey: i.PublicKey,
		CurverId: &math2.CurveID{
			Id: uint64(i.Curve),
		},
	}, nil
}

func (i *IdemixIssuerPublicKey) FromProtos(s *pp.IdemixIssuerPublicKey) error {
	if s.PublicKey == nil {
		return errors.New("invalid idemix public key, it is nil")
	}
	i.PublicKey = s.PublicKey
	if s.CurverId == nil {
		return errors.New("invalid idemix issuer public key")
	}
	i.Curve = mathlib.CurveID(s.CurverId.Id)
	return nil
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
	IdemixIssuerPublicKeys []*IdemixIssuerPublicKey
	// Auditor is the public key of the auditor.
	Auditor driver.Identity
	// IssuerIDs is a list of public keys of the entities that can issue tokens.
	IssuerIDs []driver.Identity
	// MaxToken is the maximum quantity a token can hold
	MaxToken uint64
	// QuantityPrecision is the precision used to represent quantities
	QuantityPrecision uint64
}

func NewPublicParamsFromBytes(raw []byte, label string) (*PublicParams, error) {
	pp := &PublicParams{}
	pp.Label = label
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed parsing public parameters")
	}
	return pp, nil
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
		IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{
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

func (p *PublicParams) Identifier() string {
	return p.Label
}

func (p *PublicParams) Version() string {
	return p.Ver
}

func (p *PublicParams) CertificationDriver() string {
	return p.Label
}

func (p *PublicParams) TokenDataHiding() bool {
	return true
}

func (p *PublicParams) GraphHiding() bool {
	return false
}

func (p *PublicParams) MaxTokenValue() uint64 {
	return p.MaxToken
}

func (p *PublicParams) Bytes() ([]byte, error) {
	return p.Serialize()
}

func (p *PublicParams) Auditors() []driver.Identity {
	if len(p.Auditor) == 0 {
		return []driver.Identity{}
	}
	return []driver.Identity{p.Auditor}
}

// Issuers returns the list of authorized issuers
func (p *PublicParams) Issuers() []driver.Identity {
	return p.IssuerIDs
}

func (p *PublicParams) Precision() uint64 {
	return p.QuantityPrecision
}

func (p *PublicParams) Serialize() ([]byte, error) {
	pg, err := utils2.ToProtoG1Slice(p.PedersenGenerators)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize public parameters")
	}
	rpp, err := p.RangeProofParams.ToProtos()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize range proof parameters")
	}
	issuers, err := protos.ToProtosSliceFunc(p.IssuerIDs, func(id driver.Identity) (*pp.Identity, error) {
		return &pp.Identity{
			Raw: id,
		}, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize issuer")
	}
	idemixIssuerPublicKeys, err := protos.ToProtosSlice[pp.IdemixIssuerPublicKey, *IdemixIssuerPublicKey](p.IdemixIssuerPublicKeys)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize idemix issuer public keys")
	}

	publicParams := &pp.PublicParameters{
		Identifier: p.Label,
		Version:    p.Ver,
		CurveId: &math2.CurveID{
			Id: uint64(p.Curve),
		},
		PedersenGenerators:     pg,
		RangeProofParams:       rpp,
		IdemixIssuerPublicKeys: idemixIssuerPublicKeys,
		Auditor: &pp.Identity{
			Raw: p.Auditor,
		},
		Issuers:           issuers,
		MaxToken:          p.MaxToken,
		QuantityPrecision: p.QuantityPrecision,
	}
	raw, err := proto.Marshal(publicParams)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&pp2.PublicParameters{
		Identifier: p.Label,
		Raw:        raw,
	})
}

func (p *PublicParams) Deserialize(raw []byte) error {
	container := &pp2.PublicParameters{}
	if err := json.Unmarshal(raw, container); err != nil {
		return err
	}
	if container.Identifier != p.Label {
		return errors.Errorf("invalid identifier, expecting [%s], got [%s]", p.Label, container.Identifier)
	}

	publicParams := &pp.PublicParameters{}
	if err := proto.Unmarshal(container.Raw, publicParams); err != nil {
		return errors.Wrapf(err, "failed unmarshalling public parameters")
	}

	p.Label = publicParams.Identifier
	p.Ver = publicParams.Version
	if publicParams.CurveId == nil {
		return errors.Errorf("invalid curve id, expecting curve id, got nil")
	}
	p.Curve = mathlib.CurveID(publicParams.CurveId.Id)
	p.MaxToken = publicParams.MaxToken
	p.QuantityPrecision = publicParams.QuantityPrecision
	pg, err := utils2.FromG1ProtoSlice(publicParams.PedersenGenerators)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize public parameters")
	}
	p.PedersenGenerators = pg
	issuers, err := protos.FromProtosSliceFunc2(publicParams.Issuers, func(id *pp.Identity) (driver.Identity, error) {
		if id == nil {
			return nil, nil
		}
		return id.Raw, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize issuers")
	}
	p.IssuerIDs = issuers

	p.IdemixIssuerPublicKeys = slices.GenericSliceOfPointers[IdemixIssuerPublicKey](len(publicParams.IdemixIssuerPublicKeys))
	err = protos.FromProtosSlice[pp.IdemixIssuerPublicKey, *IdemixIssuerPublicKey](publicParams.IdemixIssuerPublicKeys, p.IdemixIssuerPublicKeys)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize idemix issuer public keys")
	}
	if publicParams.Auditor != nil {
		p.Auditor = publicParams.Auditor.Raw
	}

	p.RangeProofParams = &RangeProofParams{}
	if err := p.RangeProofParams.FromProto(publicParams.RangeProofParams); err != nil {
		return errors.Wrapf(err, "failed to deserialize range proof parameters")
	}

	return nil
}

func (p *PublicParams) GeneratePedersenParameters() error {
	curve := mathlib.Curves[p.Curve]
	rand, err := curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	p.PedersenGenerators = make([]*mathlib.G1, 3)

	for i := 0; i < len(p.PedersenGenerators); i++ {
		p.PedersenGenerators[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return nil
}

func (p *PublicParams) GenerateRangeProofParameters(bitLength uint64) error {
	curve := mathlib.Curves[p.Curve]

	p.RangeProofParams = &RangeProofParams{
		P:              curve.HashToG1([]byte(strconv.Itoa(0))),
		Q:              curve.HashToG1([]byte(strconv.Itoa(1))),
		BitLength:      bitLength,
		NumberOfRounds: log2(bitLength),
	}
	p.RangeProofParams.LeftGenerators = make([]*mathlib.G1, bitLength)
	p.RangeProofParams.RightGenerators = make([]*mathlib.G1, bitLength)

	for i := uint64(0); i < bitLength; i++ {
		p.RangeProofParams.LeftGenerators[i] = curve.HashToG1([]byte("RangeProof." + strconv.FormatUint(2*(i+1), 10)))
		p.RangeProofParams.RightGenerators[i] = curve.HashToG1([]byte("RangeProof." + strconv.FormatUint(2*(i+1)+1, 10)))
	}

	return nil
}

func (p *PublicParams) AddAuditor(auditor driver.Identity) {
	p.Auditor = auditor
}

func (p *PublicParams) AddIssuer(id driver.Identity) {
	p.IssuerIDs = append(p.IssuerIDs, id)
}

func (p *PublicParams) ComputeHash() ([]byte, error) {
	raw, err := p.Bytes()
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

func (p *PublicParams) ComputeMaxTokenValue() uint64 {
	return 1<<p.RangeProofParams.BitLength - 1
}

func (p *PublicParams) String() string {
	res, err := json.MarshalIndent(p, " ", "  ")
	if err != nil {
		return err.Error()
	}
	return string(res)
}

func (p *PublicParams) Validate() error {
	if int(p.Curve) > len(mathlib.Curves)-1 {
		return errors.Errorf("invalid public parameters: invalid curveID [%d > %d]", int(p.Curve), len(mathlib.Curves)-1)
	}
	if len(p.IdemixIssuerPublicKeys) != 1 {
		return errors.Errorf("expected one idemix issuer public key, found [%d]", len(p.IdemixIssuerPublicKeys))
	}

	for _, issuer := range p.IdemixIssuerPublicKeys {
		if issuer == nil {
			return errors.Errorf("invalid idemix issuer public key, it is nil")
		}
		if len(issuer.PublicKey) == 0 {
			return errors.Errorf("expected idemix issuer public key to be non-empty")
		}
		if int(issuer.Curve) > len(mathlib.Curves)-1 {
			return errors.Errorf("invalid public parameters: invalid idemix curveID [%d > %d]", int(p.Curve), len(mathlib.Curves)-1)
		}
	}
	if err := math.CheckElements(p.PedersenGenerators, p.Curve, 3); err != nil {
		return errors.Wrapf(err, "invalid pedersen generators")
	}
	if p.RangeProofParams == nil {
		return errors.New("invalid public parameters: nil range proof parameters")
	}
	bitLength := p.RangeProofParams.BitLength
	supportedPrecisions := collections.NewSet(SupportedPrecisions...)
	if !supportedPrecisions.Contains(bitLength) {
		return errors.Errorf("invalid bit length [%d], should be one of [%v]", bitLength, supportedPrecisions.ToSlice())
	}
	err := p.RangeProofParams.Validate(p.Curve)
	if err != nil {
		return errors.Wrap(err, "invalid public parameters")
	}
	if p.QuantityPrecision != p.RangeProofParams.BitLength {
		return errors.Errorf("invalid public parameters: quantity precision should be [%d] instead it is [%d]", p.RangeProofParams.BitLength, p.QuantityPrecision)
	}
	maxToken := p.ComputeMaxTokenValue()
	if maxToken != p.MaxToken {
		return errors.Errorf("invalid maxt token, [%d]!=[%d]", maxToken, p.MaxToken)
	}
	// if len(pp.Issuers) == 0 {
	//	return errors.New("invalid public parameters: empty list of issuers")
	// }
	return nil
}

func log2(x uint64) uint64 {
	return 63 - uint64(bits.LeadingZeros64(x))
}
