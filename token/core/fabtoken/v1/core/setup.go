/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
	"github.com/pkg/errors"
)

const (
	// PublicParameters is the key to be used to look up fabtoken parameters
	PublicParameters = "fabtoken"
	Version          = "1.0.0"
	DefaultPrecision = uint64(64)
)

// PublicParams is the public parameters for fabtoken
type PublicParams struct {
	// Label is the label associated with the PublicParams.
	// It can be used by the driver for versioning purpose.
	Label string
	// Ver is the version of these public params
	Ver string
	// The precision of token quantities
	QuantityPrecision uint64
	// This is set when audit is enabled
	Auditor []byte
	// This encodes the list of authorized issuers
	IssuerIDs []driver.Identity
	// MaxToken is the maximum quantity a token can hold
	MaxToken uint64
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
		Ver:               Version,
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

func (p *PublicParams) Version() string {
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
	return json.Marshal(p)
}

// Serialize marshals a wrapper around PublicParams (SerializedPublicParams)
func (p *PublicParams) Serialize() ([]byte, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&pp2.PublicParameters{
		Identifier: p.Label,
		Raw:        raw,
	})
}

// Deserialize un-marshals the passed bytes into PublicParams
func (p *PublicParams) Deserialize(raw []byte) error {
	publicParams := &pp2.PublicParameters{}
	if err := json.Unmarshal(raw, publicParams); err != nil {
		return err
	}
	if publicParams.Identifier != p.Label {
		return errors.Errorf("invalid identifier, expecting 'fabtoken', got [%s]", publicParams.Raw)
	}
	return json.Unmarshal(publicParams.Raw, p)
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
	if len(p.IssuerIDs) == 0 {
		return errors.New("invalid public parameters: empty list of issuers")
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
