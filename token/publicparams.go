/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// PublicParamsFetcher models the public parameters fetcher
type PublicParamsFetcher interface {
	// Fetch fetches the public parameters from the backend
	Fetch() ([]byte, error)
}

// PublicParametersManager exposes methods to manage the public parameters
// TODO: add precision
type PublicParametersManager struct {
	ppm tokenapi.PublicParamsManager
}

// CertificationDriver return the certification driver used to certify that tokens exist
func (c *PublicParametersManager) CertificationDriver() string {
	return c.ppm.PublicParameters().CertificationDriver()
}

// GraphHiding returns true if graph hiding is enabled
func (c *PublicParametersManager) GraphHiding() bool {
	return c.ppm.PublicParameters().GraphHiding()
}

// TokenDataHiding returns true if data hiding is enabled
func (c *PublicParametersManager) TokenDataHiding() bool {
	return c.ppm.PublicParameters().TokenDataHiding()
}

// MaxTokenValue returns the maximum value a token can contain
func (c *PublicParametersManager) MaxTokenValue() uint64 {
	return c.ppm.PublicParameters().MaxTokenValue()
}

// Bytes returns the public parameters as a byte array
func (c *PublicParametersManager) Bytes() ([]byte, error) {
	return c.ppm.PublicParameters().Bytes()
}

// ForceFetch forcefully fetches the public parameters from the backend
func (c *PublicParametersManager) ForceFetch() error {
	return c.ppm.ForceFetch()
}

// Identifier returns the identifier of the public parameters
func (c *PublicParametersManager) Identifier() string {
	return c.ppm.PublicParameters().Identifier()
}

// Auditors returns the list of auditors' identities
func (c *PublicParametersManager) Auditors() []view.Identity {
	return c.ppm.PublicParameters().Auditors()
}
