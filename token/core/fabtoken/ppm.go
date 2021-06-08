/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import "github.com/hyperledger-labs/fabric-token-sdk/token/api"

type PublicParamsManager struct {
	pp *PublicParams
}

func NewPublicParamsManager(pp *PublicParams) *PublicParamsManager {
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

func (v *PublicParamsManager) PublicParameters() api.PublicParameters {
	return v.pp
}

func (v *PublicParamsManager) NewCertifierKeyPair() ([]byte, []byte, error) {
	panic("not supported")
}

func (v *PublicParamsManager) ForceFetch() error {
	// TODO: implement this
	return nil
}
