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

type PublicParamsManager struct {
	pp                 *crypto.PublicParams
	publicParamsLoader PublicParamsLoader
	mutex              sync.RWMutex
}

func New(publicParamsLoader PublicParamsLoader) *PublicParamsManager {
	return &PublicParamsManager{publicParamsLoader: publicParamsLoader, mutex: sync.RWMutex{}}
}

func NewFromParams(pp *crypto.PublicParams) (*PublicParamsManager, error) {
	if pp == nil {
		return nil, errors.New("public parameters not set")
	}
	return &PublicParamsManager{pp: pp, mutex: sync.RWMutex{}}, nil
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

func (v *PublicParamsManager) Update() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.publicParamsLoader == nil {
		return errors.New("public parameters loader not set")
	}

	pp, err := v.publicParamsLoader.FetchParams()
	if err != nil {
		return errors.WithMessagef(err, "failed updating public parameters")
	}
	v.pp = pp

	return nil
}

func (v *PublicParamsManager) Fetch() ([]byte, error) {
	logger.Debugf("fetch public parameters...")
	if v.publicParamsLoader == nil {
		return nil, errors.New("public parameters loader not set")
	}
	raw, err := v.publicParamsLoader.Fetch()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed force fetching public parameters")
	}
	logger.Debugf("fetched public parameters [%s]", hash.Hashable(raw).String())

	return raw, nil
}

func (v *PublicParamsManager) PublicParams() *crypto.PublicParams {
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
