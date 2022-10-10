/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// PublicParamsManager loads fabtoken public parameters
type PublicParamsManager struct {
	// fabtoken public parameters
	pp *PublicParams
	// a loader for fabric public parameters
	publicParamsLoader PublicParamsLoader

	mutex sync.RWMutex
}

// NewPublicParamsManager initializes a PublicParamsManager with the passed PublicParamsLoader
func NewPublicParamsManager(publicParamsLoader PublicParamsLoader) *PublicParamsManager {
	return &PublicParamsManager{publicParamsLoader: publicParamsLoader, mutex: sync.RWMutex{}}
}

// NewPublicParamsManagerFromParams initializes a PublicParamsManager with the passed PublicParams
func NewPublicParamsManagerFromParams(pp *PublicParams) *PublicParamsManager {
	if pp == nil {
		panic("public parameters must be non-nil")
	}
	return &PublicParamsManager{pp: pp, mutex: sync.RWMutex{}}
}

// PublicParameters returns the public parameters of PublicParamsManager
func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.PublicParams()
}

// SerializePublicParameters returns the public params in a serialized form
func (v *PublicParamsManager) SerializePublicParameters() ([]byte, error) {
	return v.PublicParams().Serialize()
}

// NewCertifierKeyPair returns the key pair of a certifier, in this instantiation, the method panics
// fabtoken does not support token certification
func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("NewCertifierKeyPair cannot be called from fabtoken")
}

// Update sets the public parameters of the PublicParamsManager to the public parameters
// associated with its PublicParamsLoader
func (v *PublicParamsManager) Update() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.publicParamsLoader == nil {
		return errors.New("public parameters loader not set")
	}

	pp, err := v.publicParamsLoader.FetchParams()
	if err != nil {
		return errors.WithMessagef(err, "failed force fetching public parameters")
	}
	v.pp = pp

	return nil
}

// Fetch fetches the public parameters from the backend
func (v *PublicParamsManager) Fetch() ([]byte, error) {
	if v.publicParamsLoader == nil {
		return nil, errors.New("public parameters loader not set")
	}
	raw, err := v.publicParamsLoader.Fetch()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed force fetching public parameters")
	}
	return raw, nil
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
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.pp
}

// Validate validates the public parameters
func (v *PublicParamsManager) Validate() error {
	pp := v.PublicParams()
	if pp == nil {
		return errors.New("public parameters not set")
	}
	return pp.Validate()
}
