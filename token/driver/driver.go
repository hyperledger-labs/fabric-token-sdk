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
	PPReader
	// NewTokenService returns a new TokenManagerService instance.
	NewTokenService(tmsID TMSID, publicParams []byte) (TokenManagerService, error)
	// NewDefaultValidator returns a new Validator instance from the passed public parameters
	NewDefaultValidator(pp PublicParameters) (Validator, error)
}
