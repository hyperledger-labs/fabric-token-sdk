/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// PublicParamsManager loads fabtoken public parameters
type PublicParamsManager struct {
	// fabtoken public parameters
	pp *PublicParams
	// a loader for fabric public parameters
	publicParamsLoader PublicParamsLoader
}

// NewPublicParamsManager initializes a PublicParamsManager with the passed PublicParamsLoader
func NewPublicParamsManager(publicParamsLoader PublicParamsLoader) *PublicParamsManager {
	return &PublicParamsManager{publicParamsLoader: publicParamsLoader}
}

// NewPublicParamsManagerFromParams initializes a PublicParamsManager with the passed PublicParams
func NewPublicParamsManagerFromParams(pp *PublicParams) *PublicParamsManager {
	if pp == nil {
		panic("public parameters must be non-nil")
	}
	return &PublicParamsManager{pp: pp}
}

// PublicParameters returns the public parameters of PublicParamsManager
func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.PublicParams()
}

// NewCertifierKeyPair returns the key pair of a certifier, in this instantiation, the method panics
// fabtoken does not support token certification
func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("NewCertifierKeyPair cannot be called from fabtoken")
}

// ForceFetch sets the public parameters of the PublicParamsManager to the public parameters
// associated with its PublicParamsLoader
func (v *PublicParamsManager) ForceFetch() error {
	if v.publicParamsLoader == nil {
		return errors.New("public parameters loader not set")
	}

	pp, err := v.publicParamsLoader.ForceFetch()
	if err != nil {
		return errors.WithMessagef(err, "failed force fetching public parameters")
	}
	v.pp = pp

	return nil
}

// AuditorIdentity returns the identity of the auditor
func (v *PublicParamsManager) AuditorIdentity() view.Identity {
	return v.PublicParams().Auditor
}

// Issuers returns the array of admissible issuers
func (v *PublicParamsManager) Issuers() [][]byte {
	return v.PublicParams().Issuers
}

// PublicParams returns the fabtoken public parameters
func (v *PublicParamsManager) PublicParams() *PublicParams {
	return v.pp
}
