/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// PPMFactory is a factory for creating zkatdlog public parameters managers.
type PPMFactory struct{ *Base }

// NewPPMFactory returns a new factory for the zkatdlog public parameters manager.
func NewPPMFactory() core.NamedFactory[driver.PPMFactory] {
	return core.NamedFactory[driver.PPMFactory]{
		Name:   core.DriverIdentifier(v1.DLogNoGHDriverName, v1.ProtocolV1),
		Driver: &PPMFactory{},
	}
}

// NewValidatorDriver returns a new factory for the zkatdlog validator driver.
func NewValidatorDriver() core.NamedFactory[driver.ValidatorDriver] {
	return core.NamedFactory[driver.ValidatorDriver]{
		Name:   core.DriverIdentifier(v1.DLogNoGHDriverName, v1.ProtocolV1),
		Driver: &PPMFactory{},
	}
}

// NewPublicParametersManager returns a new zkatdlog public parameters manager for the passed public parameters.
func (d *PPMFactory) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*v1.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	return common.NewPublicParamsManagerFromParams[*v1.PublicParams](pp)
}
