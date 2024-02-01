/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// ServiceProvider is used to return instances of a given type
type ServiceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

// Driver is the interface that must be implemented by a token driver.
type Driver interface {
	// PublicParametersFromBytes unmarshals the bytes to a PublicParameters instance.
	PublicParametersFromBytes(params []byte) (PublicParameters, error)
	// NewTokenService returns a new TokenManagerService instance.
	NewTokenService(sp ServiceProvider, networkID string, channel string, namespace string, publicParams []byte) (TokenManagerService, error)
	// NewPublicParametersManager returns a new PublicParametersManager instance from the passed public parameters
	NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error)
	// NewValidator returns a new Validator instance from the passed public parameters
	NewValidator(pp PublicParameters) (Validator, error)
}

// ExtendedDriver is the interface that models additional services a token driver may offer
type ExtendedDriver interface {
	// NewWalletService returns an instance of the WalletService interface for the passed arguments
	NewWalletService(sp ServiceProvider, networkID string, channel string, namespace string, params PublicParameters) (WalletService, error)
}
