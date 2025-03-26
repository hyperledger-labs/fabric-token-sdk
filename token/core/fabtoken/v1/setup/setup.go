/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package setup

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	encoding "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/pp"
	fabpp "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/protos"
	"github.com/pkg/errors"
)

const (
	// PublicParameters is the key to be used to look up fabtoken parameters
	PublicParameters = "fabtoken"
	ProtocolV1       = uint64(1)
	DefaultPrecision = uint64(64)
)

// PublicParams is the public parameters for fabtoken
type PublicParams struct {
	// Label is the label associated with the PublicParams.
	// It can be used by the driver for versioning purpose.
	Label string
	// Ver is the version of these public params
	Ver uint64
	// The precision of token quantities
	QuantityPrecision uint64
	// MaxToken is the maximum quantity a token can hold
	MaxToken uint64
	// This is set when audit is enabled
	Auditor []byte
	// This encodes the list of authorized issuers
	IssuerIDs []driver.Identity
}

// Setup initializes PublicParams
func Setup(precision uint64) (*PublicParams, error) {
	if precision > 64 {
		return nil, errors.Errorf("invalid precision [%d], must be smaller or equal than 64", precision)
	}
	if precision == 0 {
		return nil, errors.New("invalid precision, should be greater than 0")
	}
	return &PublicParams{
		Label:             PublicParameters,
		Ver:               ProtocolV1,
		QuantityPrecision: precision,
		MaxToken:          uint64(1<<precision) - 1,
	}, nil
}

// NewPublicParamsFromBytes deserializes the raw bytes into public parameters
// The resulting public parameters are labeled with the passed label
func NewPublicParamsFromBytes(raw []byte, label string) (*PublicParams, error) {
	pp := &PublicParams{}
	pp.Label = label
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

// Identifier returns the label associated with the PublicParams
// todo shall we used Identifier instead of Label?
func (p *PublicParams) Identifier() string {
	return p.Label
}

func (p *PublicParams) Version() uint64 {
	return p.Ver
}

// TokenDataHiding indicates if the PublicParams corresponds to a driver that hides token data
// fabtoken does not hide token data, hence, TokenDataHiding returns false
func (p *PublicParams) TokenDataHiding() bool {
	return false
}

// CertificationDriver returns the label of the PublicParams
// From the label, one can deduce what certification process will be used if any.
func (p *PublicParams) CertificationDriver() string {
	return p.Label
}

// GraphHiding indicates if the PublicParams corresponds to a driver that hides the transaction graph
// fabtoken does not hide the graph, hence, GraphHiding returns false
func (p *PublicParams) GraphHiding() bool {
	return false
}

// MaxTokenValue returns the maximum value that a token can hold according to PublicParams
func (p *PublicParams) MaxTokenValue() uint64 {
	return p.MaxToken
}

// Bytes marshals PublicParams
func (p *PublicParams) Bytes() ([]byte, error) {
	issuers, err := protos.ToProtosSliceFunc(p.IssuerIDs, func(id driver.Identity) (*fabpp.Identity, error) {
		return &fabpp.Identity{
			Raw: id,
		}, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize issuer")
	}

	pp := &fabpp.PublicParameters{
		Identifier: p.Label,
		Version:    p.Ver,
		Auditor: &fabpp.Identity{
			Raw: p.Auditor,
		},
		Issuers:           issuers,
		MaxToken:          p.MaxToken,
		QuantityPrecision: p.QuantityPrecision,
	}
	return proto.Marshal(p)
}

func (p *PublicParams) FromBytes(data []byte) error {
	publicParams := &fabpp.PublicParameters{}
	if err := proto.Unmarshal(data, publicParams); err != nil {
		return errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	p.Ver = publicParams.Version
	p.QuantityPrecision = publicParams.QuantityPrecision
	p.MaxToken = publicParams.MaxToken
	issuers, err := protos.FromProtosSliceFunc2(publicParams.Issuers, func(id *fabpp.Identity) (driver.Identity, error) {
		if id == nil {
			return nil, nil
		}
		return id.Raw, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize issuers")
	}
	p.IssuerIDs = issuers
	if publicParams.Auditor != nil {
		p.Auditor = publicParams.Auditor.Raw
	}
	return nil
}

// Serialize marshals a wrapper around PublicParams (SerializedPublicParams)
func (p *PublicParams) Serialize() ([]byte, error) {
	raw, err := p.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize public parameters")
	}
	return encoding.Marshal(&pp.PublicParameters{
		Identifier: p.Label,
		Raw:        raw,
	})
}

// Deserialize un-marshals the passed bytes into PublicParams
func (p *PublicParams) Deserialize(raw []byte) error {
	container, err := encoding.Unmarshal(raw)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize public parameters")
	}
	if container.Identifier != p.Label {
		return errors.Errorf("invalid identifier, expecting 'fabtoken', got [%s]", container.Identifier)
	}
	return p.FromBytes(container.Raw)
}

// AuditorIdentity returns the auditor identity encoded in PublicParams
func (p *PublicParams) AuditorIdentity() driver.Identity {
	return p.Auditor
}

// AddAuditor sets the Auditor field in PublicParams to the passed identity
func (p *PublicParams) AddAuditor(auditor driver.Identity) {
	p.Auditor = auditor
}

// AddIssuer adds the passed issuer to the array of Issuers in PublicParams
func (p *PublicParams) AddIssuer(issuer driver.Identity) {
	p.IssuerIDs = append(p.IssuerIDs, issuer)
}

// SetIssuers sets the issuers to the passed identities
func (p *PublicParams) SetIssuers(ids []driver.Identity) {
	p.IssuerIDs = ids
}

// SetAuditors sets the auditors to the passed identities
func (p *PublicParams) SetAuditors(ids []driver.Identity) {
	p.Auditor = ids[0]
}

// Auditors returns the list of authorized auditors
// fabtoken only supports a single auditor
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

// Precision returns the quantity precision encoded in PublicParams
func (p *PublicParams) Precision() uint64 {
	return p.QuantityPrecision
}

// Validate validates the public parameters
func (p *PublicParams) Validate() error {
	if p.QuantityPrecision > 64 {
		return errors.Errorf("invalid precision [%d], must be less than 64", p.QuantityPrecision)
	}
	if p.QuantityPrecision == 0 {
		return errors.New("invalid precision, must be greater than 0")
	}
	maxTokenValue := uint64(1<<p.Precision()) - 1
	if p.MaxToken > maxTokenValue {
		return errors.Errorf("max token value is invalid [%d]>[%d]", p.MaxToken, maxTokenValue)
	}
	return nil
}

func (p *PublicParams) String() string {
	res, err := json.MarshalIndent(p, " ", "  ")
	if err != nil {
		return err.Error()
	}
	return string(res)
}
