/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
	"github.com/pkg/errors"
)

const (
	// PublicParameters is the key to be used to look up fabtoken parameters
	PublicParameters = "fabtoken"
	DefaultPrecision = uint64(64)
)

// PublicParams is the public parameters for fabtoken
type PublicParams struct {
	// Label is the label associated with the PublicParams.
	// It can be used by the driver for versioning purpose.
	Label string
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
func (pp *PublicParams) Identifier() string {
	return pp.Label
}

// TokenDataHiding indicates if the PublicParams corresponds to a driver that hides token data
// fabtoken does not hide token data, hence, TokenDataHiding returns false
func (pp *PublicParams) TokenDataHiding() bool {
	return false
}

// CertificationDriver returns the label of the PublicParams
// From the label, one can deduce what certification process will be used if any.
func (pp *PublicParams) CertificationDriver() string {
	return pp.Label
}

// GraphHiding indicates if the PublicParams corresponds to a driver that hides the transaction graph
// fabtoken does not hide the graph, hence, GraphHiding returns false
func (pp *PublicParams) GraphHiding() bool {
	return false
}

// MaxTokenValue returns the maximum value that a token can hold according to PublicParams
func (pp *PublicParams) MaxTokenValue() uint64 {
	return pp.MaxToken
}

// Bytes marshals PublicParams
func (pp *PublicParams) Bytes() ([]byte, error) {
	return json.Marshal(pp)
}

// Serialize marshals a wrapper around PublicParams (SerializedPublicParams)
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

// Deserialize un-marshals the passed bytes into PublicParams
func (pp *PublicParams) Deserialize(raw []byte) error {
	publicParams := &pp2.PublicParameters{}
	if err := json.Unmarshal(raw, publicParams); err != nil {
		return err
	}
	if publicParams.Identifier != pp.Label {
		return errors.Errorf("invalid identifier, expecting 'fabtoken', got [%s]", publicParams.Raw)
	}
	return json.Unmarshal(publicParams.Raw, pp)
}

// AuditorIdentity returns the auditor identity encoded in PublicParams
func (pp *PublicParams) AuditorIdentity() driver.Identity {
	return pp.Auditor
}

// AddAuditor sets the Auditor field in PublicParams to the passed identity
func (pp *PublicParams) AddAuditor(auditor driver.Identity) {
	pp.Auditor = auditor
}

// AddIssuer adds the passed issuer to the array of Issuers in PublicParams
func (pp *PublicParams) AddIssuer(issuer driver.Identity) {
	pp.IssuerIDs = append(pp.IssuerIDs, issuer)
}

// Auditors returns the list of authorized auditors
// fabtoken only supports a single auditor
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

// Precision returns the quantity precision encoded in PublicParams
func (pp *PublicParams) Precision() uint64 {
	return pp.QuantityPrecision
}

// Validate validates the public parameters
func (pp *PublicParams) Validate() error {
	if pp.QuantityPrecision > 64 {
		return errors.Errorf("invalid precision [%d], must be less than 64", pp.QuantityPrecision)
	}
	if pp.QuantityPrecision == 0 {
		return errors.New("invalid precision, must be greater than 0")
	}
	maxTokenValue := uint64(1<<pp.Precision()) - 1
	if pp.MaxToken > maxTokenValue {
		return errors.Errorf("max token value is invalid [%d]>[%d]", pp.MaxToken, maxTokenValue)
	}
	return nil
}

func (pp *PublicParams) String() string {
	res, err := json.MarshalIndent(pp, " ", "  ")
	if err != nil {
		return err.Error()
	}
	return string(res)
}
