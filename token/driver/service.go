/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/pkg/errors"
)

type NamedFactory[T any] struct {
	Name   TokenDriverName
	Driver T
}

type factoryDirectory[T PPReader] struct {
	factories map[TokenDriverName]T
}

func newFactoryDirectory[T PPReader](fs ...NamedFactory[T]) *factoryDirectory[T] {
	factories := make(map[TokenDriverName]T, len(fs))
	for _, f := range fs {
		factories[f.Name] = f.Driver
	}
	return &factoryDirectory[T]{factories: factories}
}

// PublicParametersFromBytes unmarshals the bytes to a driver.PublicParameters instance.
// The passed bytes are expected to encode a driver.SerializedPublicParameters instance.
// If no driver is registered for the public params' identifier, it returns an error.
func (s *factoryDirectory[T]) PublicParametersFromBytes(params []byte) (PublicParameters, error) {
	pp, err := serializedPublicParametersFromBytes(params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling public params")
	}

	if f, ok := s.factories[TokenDriverName(pp.Identifier)]; ok {
		return f.PublicParametersFromBytes(params)
	}
	return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier)
}

// serializedPublicParametersFromBytes returns a driver.SerializedPublicParameters instance from the passed bytes.
func serializedPublicParametersFromBytes(raw []byte) (*SerializedPublicParameters, error) {
	pp := &SerializedPublicParameters{}
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing public parameters")
	}
	return pp, nil
}

type PPManagerFactoryService struct {
	*factoryDirectory[PPMFactory]
}

func NewPPManagerFactoryService(instantiators ...NamedFactory[PPMFactory]) *PPManagerFactoryService {
	return &PPManagerFactoryService{factoryDirectory: newFactoryDirectory(instantiators...)}
}

// NewPublicParametersManager returns a new instance of driver.PublicParamsManager for the passed parameters.
// If no driver is registered for the public params' identifier, it returns an error
func (s *PPManagerFactoryService) NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error) {
	if instantiator, ok := s.factories[TokenDriverName(pp.Identifier())]; ok {
		return instantiator.NewPublicParametersManager(pp)
	}
	return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
}

func (s *PPManagerFactoryService) DefaultValidator(pp PublicParameters) (Validator, error) {
	if instantiator, ok := s.factories[TokenDriverName(pp.Identifier())]; ok {
		return instantiator.DefaultValidator(pp)
	}
	return nil, errors.Errorf("cannot load default validator, driver [%s] not found", pp.Identifier())
}

type WalletServiceFactoryService struct {
	*factoryDirectory[WalletServiceFactory]
}

func NewWalletServiceFactoryService(fs ...NamedFactory[WalletServiceFactory]) *WalletServiceFactoryService {
	return &WalletServiceFactoryService{factoryDirectory: newFactoryDirectory(fs...)}
}

func (s *WalletServiceFactoryService) NewWalletService(tmsConfig Config, ppRaw []byte) (WalletService, error) {
	pp, err := s.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, err
	}
	if factory, ok := s.factories[TokenDriverName(pp.Identifier())]; ok {
		return factory.NewWalletService(tmsConfig, pp)
	}
	return nil, errors.Errorf("no validator found for token driver [%s]", pp.Identifier())
}

type TokenDriverService struct {
	*factoryDirectory[Driver]
}

func NewTokenDriverService(factories []NamedFactory[Driver]) *TokenDriverService {
	return &TokenDriverService{factoryDirectory: newFactoryDirectory(factories...)}
}

func (s *TokenDriverService) NewTokenService(tmsID TMSID, publicParams []byte) (TokenManagerService, error) {
	pp, err := s.PublicParametersFromBytes(publicParams)
	if err != nil {
		return nil, err
	}
	if driver, ok := s.factories[TokenDriverName(pp.Identifier())]; ok {
		return driver.NewTokenService(tmsID, publicParams)
	}
	return nil, errors.Errorf("no token driver named '%s' found", TokenDriverName(pp.Identifier()))
}

func (s *TokenDriverService) NewDefaultValidator(pp PublicParameters) (Validator, error) {
	if driver, ok := s.factories[TokenDriverName(pp.Identifier())]; ok {
		return driver.NewDefaultValidator(pp)
	}
	return nil, errors.Errorf("no validator found for token driver [%s]", pp.Identifier())
}

var managerType = &TokenDriverService{}

func GetTokenDriverService(sp ServiceProvider) (*TokenDriverService, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token driver service")
	}
	return s.(*TokenDriverService), nil
}
