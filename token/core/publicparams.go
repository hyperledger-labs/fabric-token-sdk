/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
	"github.com/pkg/errors"
)

func PublicParametersFromBytes(params []byte) (api.PublicParameters, error) {
	pp, err := SerializedPublicParametersFromBytes(params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling public params")
	}

	d, ok := drivers[pp.Identifier]
	if !ok {
		return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier)
	}
	return d.PublicParametersFromBytes(params)
}

func SerializedPublicParametersFromBytes(raw []byte) (*api.SerializedPublicParameters, error) {
	pp := &api.SerializedPublicParameters{}
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing public parameters")
	}
	return pp, nil
}

func NewPublicParametersManager(pp api.PublicParameters) (api.PublicParamsManager, error) {
	d, ok := drivers[pp.Identifier()]
	if !ok {
		return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
	}
	return d.NewPublicParametersManager(pp)
}
