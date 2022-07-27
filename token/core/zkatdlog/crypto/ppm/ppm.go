/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ppm

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog")

type PublicParamsLoader interface {
	Load() (*crypto.PublicParams, error)
	ForceFetch() (*crypto.PublicParams, error)
}

type PublicParamsManager struct {
	pp                 *crypto.PublicParams
	publicParamsLoader PublicParamsLoader
}

func New(publicParamsLoader PublicParamsLoader) *PublicParamsManager {
	return &PublicParamsManager{publicParamsLoader: publicParamsLoader}
}

func NewFromParams(pp *crypto.PublicParams) (*PublicParamsManager, error) {
	if pp == nil {
		return nil, errors.New("public parameters not set")
	}
	return &PublicParamsManager{pp: pp}, nil
}

func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.PublicParams()
}

func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("not supported")
}

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

func (v *PublicParamsManager) PublicParams() *crypto.PublicParams {
	return v.pp
}
