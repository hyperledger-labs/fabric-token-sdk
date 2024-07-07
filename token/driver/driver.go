/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type Config interface {
	ID() TMSID
	TranslatePath(path string) string
	UnmarshalKey(key string, rawVal interface{}) error
}

// ServiceProvider is used to return instances of a given type
type ServiceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

type TokenDriverName string

type PPReader interface {
	// PublicParametersFromBytes unmarshals the bytes to a PublicParameters instance.
	PublicParametersFromBytes(params []byte) (PublicParameters, error)
}

// PPMFactory contains the static logic of the driver
type PPMFactory interface {
	PPReader
	// NewPublicParametersManager returns a new PublicParametersManager instance from the passed public parameters
	NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error)
	// DefaultValidator returns a new Validator instance from the passed public parameters
	DefaultValidator(pp PublicParameters) (Validator, error)
}

type WalletServiceFactory interface {
	PPReader
	// NewWalletService returns an instance of the WalletService interface for the passed arguments
	NewWalletService(tmsConfig Config, params PublicParameters) (WalletService, error)
}

// Driver is the interface that must be implemented by a token driver.
type Driver interface {
	PPReader
	// NewTokenService returns a new TokenManagerService instance.
	NewTokenService(sp ServiceProvider, networkID string, channel string, namespace string, publicParams []byte) (TokenManagerService, error)
	// NewValidator returns a new Validator instance from the passed public parameters
	NewValidator(sp ServiceProvider, tmsID TMSID, pp PublicParameters) (Validator, error)
}
