/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ppm

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.fabtoken")

type Vault interface {
	// PublicParams returns the public parameters
	PublicParams() ([]byte, error)
}

// PublicParamsManager loads fabtoken public parameters
type PublicParamsManager struct {
	// fabtoken public parameters
	PP *fabtoken.PublicParams
	// the vault
	Vault Vault
	// label of the public params
	PPLabel string

	Mutex sync.RWMutex
}

// NewPublicParamsManager initializes a PublicParamsManager with the passed PublicParamsLoader
func NewPublicParamsManager(PPLabel string, vault Vault) *PublicParamsManager {
	return &PublicParamsManager{
		PPLabel: PPLabel,
		Vault:   vault,
	}
}

// NewPublicParamsManagerFromParams initializes a PublicParamsManager with the passed PublicParams
func NewPublicParamsManagerFromParams(pp *fabtoken.PublicParams) (*PublicParamsManager, error) {
	if pp == nil {
		return nil, errors.Errorf("public parameters must be non-nil")
	}
	return &PublicParamsManager{PP: pp, PPLabel: pp.Label}, nil
}

// PublicParameters returns the public parameters of PublicParamsManager
func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	pp := v.PublicParams()
	if pp == nil {
		return nil
	}
	return pp
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

// Load sets the public parameters of the PublicParamsManager to the public parameters
// associated with its PublicParamsLoader
func (v *PublicParamsManager) Load() error {
	ppRaw, err := v.Vault.PublicParams()
	if err != nil {
		return errors.WithMessage(err, "failed to fetch public params from vault")
	}
	if len(ppRaw) == 0 {
		return nil
	}

	logger.Debugf("fetched public parameters [%s], unmarshal them...", hash.Hashable(ppRaw).String())
	return v.SetPublicParameters(ppRaw)
}

// SetPublicParameters updates the public parameters with the passed value
func (v *PublicParamsManager) SetPublicParameters(raw []byte) error {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	if len(raw) == 0 {
		return errors.Errorf("empty public parameters")
	}

	pp, err := fabtoken.NewPublicParamsFromBytes(raw, v.PPLabel)
	if err != nil {
		return err
	}

	if err := pp.Validate(); err != nil {
		return errors.WithMessage(err, "invalid public parameters")
	}

	v.PP = pp
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
func (v *PublicParamsManager) PublicParams() *fabtoken.PublicParams {
	v.Mutex.RLock()
	defer v.Mutex.RUnlock()
	return v.PP
}

// Validate validates the public parameters
func (v *PublicParamsManager) Validate() error {
	pp := v.PublicParams()
	if pp == nil {
		return errors.New("public parameters not set")
	}
	return pp.Validate()
}
