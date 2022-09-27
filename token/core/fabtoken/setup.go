/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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
	Issuers [][]byte
}

// NewPublicParamsFromBytes deserializes the raw bytes into public parameters
// The resulting public parameters are labeled with the passed label
func NewPublicParamsFromBytes(raw []byte, label string) (*PublicParams, error) {
	pp := &PublicParams{}
	pp.Label = label
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed parsing public parameters")
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
	return 2 ^ pp.Precision() - 1
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
	return json.Marshal(&driver.SerializedPublicParameters{
		Identifier: pp.Label,
		Raw:        raw,
	})
}

// Deserialize un-marshals the passed bytes into PublicParams
func (pp *PublicParams) Deserialize(raw []byte) error {
	publicParams := &driver.SerializedPublicParameters{}
	if err := json.Unmarshal(raw, publicParams); err != nil {
		return err
	}
	if publicParams.Identifier != pp.Label {
		return errors.Errorf("invalid identifier, expecting 'fabtoken', got [%s]", publicParams.Identifier)
	}
	return json.Unmarshal(publicParams.Raw, pp)
}

// AuditorIdentity returns the auditor identity encoded in PublicParams
func (pp *PublicParams) AuditorIdentity() view.Identity {
	return pp.Auditor
}

// AddAuditor sets the Auditor field in PublicParams to the passed identity
func (pp *PublicParams) AddAuditor(auditor view.Identity) {
	pp.Auditor = auditor
}

// AddIssuer adds the passed issuer to the array of Issuers in PublicParams
func (pp *PublicParams) AddIssuer(issuer view.Identity) {
	pp.Issuers = append(pp.Issuers, issuer)
}

// Auditors returns the list of authorized auditors
// fabtoken only supports a single auditor
func (pp *PublicParams) Auditors() []view.Identity {
	return []view.Identity{pp.Auditor}
}

// Precision returns the quantity precision encoded in PublicParams
func (pp *PublicParams) Precision() uint64 {
	return pp.QuantityPrecision
}

// Validate validates the public parameters
func (pp *PublicParams) Validate() error {
	return nil
}

// Setup initializes PublicParams
func Setup() (*PublicParams, error) {
	return &PublicParams{
		Label:             PublicParameters,
		QuantityPrecision: DefaultPrecision,
	}, nil
}
