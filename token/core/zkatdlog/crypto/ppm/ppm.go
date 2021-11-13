/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ppm

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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

func NewFromParams(pp *crypto.PublicParams) *PublicParamsManager {
	if pp == nil {
		panic("public parameters must be non-nil")
	}
	return &PublicParamsManager{pp: pp}
}

func (v *PublicParamsManager) SetAuditor(auditor []byte) ([]byte, error) {
	identityDeserializer := &x509.MSPIdentityDeserializer{}
	_, err := identityDeserializer.DeserializeVerifier(auditor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve auditor's identity")
	}

	pp := v.PublicParams()
	raw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	pp = &crypto.PublicParams{}
	pp.Label = v.pp.Label
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
	i := &math.G1{}
	err := json.Unmarshal(bytes, i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add new AnonymousIssuer")
	}

	raw, err := v.PublicParams().Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize public parameters")
	}

	return raw, nil
}

func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.PublicParams()
}

func (v *PublicParamsManager) SetCertifier(bytes []byte) ([]byte, error) {
	panic("SetCertifier cannot be called from zkatdlog without graph hiding")
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
