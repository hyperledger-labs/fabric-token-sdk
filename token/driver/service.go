/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/pkg/errors"
)

type TokenInstantiatorService struct {
	instantiators map[TokenDriverName]Instantiator
}

func NewTokenInstantiatorService(instantiators ...NamedInstantiator) *TokenInstantiatorService {
	is := make(map[TokenDriverName]Instantiator, len(instantiators))
	for _, instantiator := range instantiators {
		is[instantiator.Name] = instantiator.Instantiator
	}
	return &TokenInstantiatorService{instantiators: is}
}

// PublicParametersFromBytes unmarshals the bytes to a driver.PublicParameters instance.
// The passed bytes are expected to encode a driver.SerializedPublicParameters instance.
// If no driver is registered for the public params' identifier, it returns an error.
func (s *TokenInstantiatorService) PublicParametersFromBytes(params []byte) (PublicParameters, error) {
	pp, err := serializedPublicParametersFromBytes(params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling public params")
	}

	if instantiator, ok := s.instantiators[TokenDriverName(pp.Identifier)]; ok {
		return instantiator.PublicParametersFromBytes(params)
	}
	return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier)
}

// NewPublicParametersManager returns a new instance of driver.PublicParamsManager for the passed parameters.
// If no driver is registered for the public params' identifier, it returns an error
func (s *TokenInstantiatorService) NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error) {
	if instantiator, ok := s.instantiators[TokenDriverName(pp.Identifier())]; ok {
		return instantiator.NewPublicParametersManager(pp)
	}
	return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
}

func (s *TokenInstantiatorService) DefaultValidator(pp PublicParameters) (Validator, error) {
	if instantiator, ok := s.instantiators[TokenDriverName(pp.Identifier())]; ok {
		return instantiator.DefaultValidator(pp)
	}
	return nil, errors.Errorf("cannot load default validator, driver [%s] not found", pp.Identifier())
}

func NewTokenDriverService(drivers []NamedDriver) *TokenDriverService {
	ds := make(map[TokenDriverName]Driver, len(drivers))
	is := make(map[TokenDriverName]Instantiator, len(drivers))
	for _, d := range drivers {
		ds[d.Name] = d.Driver
		is[d.Name] = d.Driver
	}
	return &TokenDriverService{
		drivers: ds,
		TokenInstantiatorService: &TokenInstantiatorService{
			instantiators: is,
		},
	}
}

type TokenDriverService struct {
	*TokenInstantiatorService
	drivers map[TokenDriverName]Driver
}

func (s *TokenDriverService) NewTokenService(tmsID TMSID, publicParams []byte) (TokenManagerService, error) {
	pp, err := serializedPublicParametersFromBytes(publicParams)
	if err != nil {
		return nil, err
	}
	if driver, ok := s.drivers[TokenDriverName(pp.Identifier)]; ok {
		return driver.NewTokenService(nil, tmsID.Network, tmsID.Channel, tmsID.Namespace, publicParams)
	}
	return nil, errors.Errorf("no token driver named '%s' found", TokenDriverName(pp.Identifier))
}
func (s *TokenDriverService) NewValidator(tmsID TMSID, pp PublicParameters) (Validator, error) {
	if driver, ok := s.drivers[TokenDriverName(pp.Identifier())]; ok {
		return driver.NewValidator(nil, tmsID, pp)
	}
	return nil, errors.Errorf("no validator found for token driver [%s]", pp.Identifier())
}
func (s *TokenDriverService) NewWalletService(tmsID TMSID, pp PublicParameters) (WalletService, error) {
	if driver, ok := s.drivers[TokenDriverName(pp.Identifier())]; ok {
		if extendedDriver, ok := driver.(ExtendedDriver); ok {
			return extendedDriver.NewWalletService(nil, tmsID.Network, tmsID.Channel, tmsID.Namespace, pp)
		}
	}
	return nil, errors.Errorf("no validator found for token driver [%s]", pp.Identifier())
}

// serializedPublicParametersFromBytes returns a driver.SerializedPublicParameters instance from the passed bytes.
func serializedPublicParametersFromBytes(raw []byte) (*SerializedPublicParameters, error) {
	pp := &SerializedPublicParameters{}
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing public parameters")
	}
	return pp, nil
}

var managerType = &TokenDriverService{}

func GetTokenDriverService(sp ServiceProvider) (*TokenDriverService, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token driver service")
	}
	return s.(*TokenDriverService), nil
}
