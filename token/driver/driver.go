/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

// Driver is the interface that must be implemented by a token driver.
type Driver interface {
	// PublicParametersFromBytes unmarshals the bytes to a PublicParameters instance.
	PublicParametersFromBytes(params []byte) (PublicParameters, error)
	// NewTokenService returns a new TokenManagerService instance.
	NewTokenService(sp view.ServiceProvider, publicParamsFetcher PublicParamsFetcher, network string, channel string, namespace string) (TokenManagerService, error)
	// NewPublicParametersManager returns a new PublicParametersManager instance from the passed public parameters
	NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error)
	// NewValidator returns a new Validator instance from the passed public parameters
	NewValidator(pp PublicParameters) (Validator, error)
}

// ExtendedDriver is the interface that models additional services a token driver may offer
type ExtendedDriver interface {
	// NewWalletService returns an instance of the WalletService interface for the passed arguments
	NewWalletService(sp view.ServiceProvider, network string, channel string, namespace string, params PublicParameters) (WalletService, error)
}
