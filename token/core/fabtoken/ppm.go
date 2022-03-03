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

type PublicParamsManager struct {
	pp                 *PublicParams
	publicParamsLoader PublicParamsLoader
}

func NewPublicParamsManager(publicParamsLoader PublicParamsLoader) *PublicParamsManager {
	return &PublicParamsManager{publicParamsLoader: publicParamsLoader}
}

func NewPublicParamsManagerFromParams(pp *PublicParams) *PublicParamsManager {
	if pp == nil {
		panic("public parameters must be non-nil")
	}
	return &PublicParamsManager{pp: pp}
}

func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.PublicParams()
}

func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("NewCertifierKeyPair cannot be called from fabtoken")
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

func (v *PublicParamsManager) AuditorIdentity() view.Identity {
	return v.PublicParams().Auditor
}

func (v *PublicParamsManager) Issuers() [][]byte {
	return v.PublicParams().Issuers
}

func (v *PublicParamsManager) PublicParams() *PublicParams {
	if v.pp == nil {
		if v.publicParamsLoader == nil {
			panic("public parameters loaded not set")
		}
		var err error
		v.pp, err = v.publicParamsLoader.Load()
		if err != nil {
			panic(err)
		}
	}
	return v.pp
}
