/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// PPMFactory contains the static logic of the driver
type PPMFactory struct{ *base }

func NewPPMFactory() core.NamedFactory[driver.PPMFactory] {
	return core.NamedFactory[driver.PPMFactory]{
		Name:   fabtoken.PublicParameters,
		Driver: &PPMFactory{},
	}
}

func (d *PPMFactory) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return common.NewPublicParamsManagerFromParams[*fabtoken.PublicParams](pp)
}
