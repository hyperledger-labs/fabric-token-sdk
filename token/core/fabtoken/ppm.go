/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
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
	return &PublicParamsManager{pp: pp}
}

func (v *PublicParamsManager) SetAuditor(auditor []byte) ([]byte, error) {
	raw, err := v.pp.Serialize()
	if err != nil {
		return nil, err
	}
	pp := &PublicParams{}
	if err := pp.Deserialize(raw); err != nil {
		return nil, err
	}
	pp.Auditor = auditor

	raw, err = pp.Serialize()
	if err != nil {
		return nil, err
	}
	v.pp = pp
	return raw, nil
}

func (v *PublicParamsManager) AddIssuer(bytes []byte) ([]byte, error) {
	panic("implement me")
}

func (v *PublicParamsManager) SetCertifier(bytes []byte) ([]byte, error) {
	panic("SetCertifier cannot be called from fabtoken")
}

func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.pp
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

func (v *PublicParamsManager) publicParams() *PublicParams {
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
