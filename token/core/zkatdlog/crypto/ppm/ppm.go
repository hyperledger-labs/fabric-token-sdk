/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ppm

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog")

type PublicParamsManager struct {
	pp *crypto.PublicParams
}

func New(pp *crypto.PublicParams) *PublicParamsManager {
	return &PublicParamsManager{pp: pp}
}

func (v *PublicParamsManager) SetAuditor(auditor []byte) ([]byte, error) {
	identityDeserializer := &fabric.MSPX509IdentityDeserializer{}
	_, err := identityDeserializer.GetVerifier(auditor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve auditor's identity")
	}
	v.pp.Auditor = auditor
	raw, err := v.pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize public parameters")
	}
	return raw, nil
}

func (v *PublicParamsManager) AddIssuer(bytes []byte) ([]byte, error) {
	i := &bn256.G1{}
	err := json.Unmarshal(bytes, i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add new AnonymousIssuer")
	}

	raw, err := v.pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize public parameters")
	}

	return raw, nil
}

func (v *PublicParamsManager) PublicParameters() driver.PublicParameters {
	return v.pp
}

func (v *PublicParamsManager) SetCertifier(bytes []byte) ([]byte, error) {
	panic("SetCertifier cannot be called from zkatdlog without graph hiding")
}

func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("not supported")
}

func (v *PublicParamsManager) ForceFetch() error {
	// TODO: implement this
	return nil
}
