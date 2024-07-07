/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type PPMFactory struct{ *base }

func NewPPMFactory() driver.NamedFactory[driver.PPMFactory] {
	return driver.NamedFactory[driver.PPMFactory]{
		Name:   crypto.DLogPublicParameters,
		Driver: &PPMFactory{},
	}
}

func (d *PPMFactory) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return common.NewPublicParamsManagerFromParams[*crypto.PublicParams](pp)
}
