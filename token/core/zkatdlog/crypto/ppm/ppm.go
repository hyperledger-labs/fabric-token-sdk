/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ppm

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.zkatdlog")

type PublicParamsLoader interface {
	// Fetch fetches the public parameters from the backend
	Fetch() ([]byte, error)
	// FetchParams fetches the public parameters from the backend and unmarshal them
	FetchParams() (*crypto.PublicParams, error)
}

type Vault interface {
	// PublicParams returns the public parameters
	PublicParams() ([]byte, error)
}

type PublicParamsManager struct {
	PP                 *crypto.PublicParams
	PublicParamsLoader PublicParamsLoader
	// the vault
	Vault Vault
	// label of the public params
	PPLabel string

	Mutex sync.RWMutex
}

func NewPublicParamsManager(PPLabel string, vault Vault, publicParamsLoader PublicParamsLoader) *PublicParamsManager {
	return &PublicParamsManager{PPLabel: PPLabel, Vault: vault, PublicParamsLoader: publicParamsLoader, Mutex: sync.RWMutex{}}
}

func NewFromParams(pp *crypto.PublicParams) (*PublicParamsManager, error) {
	if pp == nil {
		return nil, errors.New("public parameters not set")
	}
	return &PublicParamsManager{PP: pp, Mutex: sync.RWMutex{}}, nil
}

func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.PublicParams()
}

// SerializePublicParameters returns the public params in a serialized form
func (v *PublicParamsManager) SerializePublicParameters() ([]byte, error) {
	return v.PublicParams().Serialize()
}

func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("not supported")
}

func (v *PublicParamsManager) Load() error {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	ppRaw, err := v.Vault.PublicParams()
	if err != nil {
		return errors.WithMessage(err, "failed to fetch public params from vault")
	}
	if len(ppRaw) == 0 {
		// fetch it, but does not fail if the fetching failed
		ppRaw, err = v.PublicParamsLoader.Fetch()
		if err != nil {
			logger.Errorf("failed to load public parameters from remote network [%s]", err)
			return nil
		}
	}

	logger.Debugf("fetched public parameters [%s], unmarshal them...", hash.Hashable(ppRaw).String())
	pp, err := crypto.NewPublicParamsFromBytes(ppRaw, v.PPLabel)
	if err != nil {
		return err
	}
	if err := pp.Validate(); err != nil {
		return errors.Wrap(err, "failed to validate public parameters")
	}
	logger.Debugf("fetched public parameters [%s], unmarshal them...done", hash.Hashable(ppRaw).String())

	v.PP = pp

	return nil
}

// SetPublicParameters updates the public parameters with the passed value
func (v *PublicParamsManager) SetPublicParameters(raw []byte) error {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	pp, err := crypto.NewPublicParamsFromBytes(raw, v.PPLabel)
	if err != nil {
		return err
	}

	if err := pp.Validate(); err != nil {
		return errors.WithMessage(err, "invalid public parameters")
	}

	v.PP = pp
	return nil
}

func (v *PublicParamsManager) Fetch() ([]byte, error) {
	logger.Debugf("fetch public parameters...")
	if v.PublicParamsLoader == nil {
		return nil, errors.New("public parameters loader not set")
	}
	raw, err := v.PublicParamsLoader.Fetch()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed force fetching public parameters")
	}
	logger.Debugf("fetched public parameters [%s]", hash.Hashable(raw).String())

	return raw, nil
}

func (v *PublicParamsManager) PublicParams() *crypto.PublicParams {
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
