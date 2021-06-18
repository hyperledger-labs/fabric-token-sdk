/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

func PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
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

func SerializedPublicParametersFromBytes(raw []byte) (*driver.SerializedPublicParameters, error) {
	pp := &driver.SerializedPublicParameters{}
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing public parameters")
	}
	return pp, nil
}

func NewPublicParametersManager(pp driver.PublicParameters) (driver.PublicParamsManager, error) {
	d, ok := drivers[pp.Identifier()]
	if !ok {
		return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
	}
	return d.NewPublicParametersManager(pp)
}
