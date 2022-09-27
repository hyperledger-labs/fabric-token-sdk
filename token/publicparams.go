/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// PublicParamsFetcher models the public parameters fetcher
type PublicParamsFetcher interface {
	// Fetch fetches the public parameters from the backend
	Fetch() ([]byte, error)
}

// PublicParametersManager exposes methods to manage the public parameters
type PublicParametersManager struct {
	ppm driver.PublicParamsManager
}

// Precision returns the precision used to represent the token quantity
func (c *PublicParametersManager) Precision() uint64 {
	return c.ppm.PublicParameters().Precision()
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

// SerializePublicParameters returns the public parameters in their serialized form
func (c *PublicParametersManager) SerializePublicParameters() ([]byte, error) {
	return c.ppm.SerializePublicParameters()
}

// Validate validates the public parameters
func (c *PublicParametersManager) Validate() error {
	return c.ppm.Validate()
}

// Update fetches the public parameters from the backend and update the local one accordingly
func (c *PublicParametersManager) Update() error {
	return c.ppm.Update()
}

// Identifier returns the identifier of the public parameters
func (c *PublicParametersManager) Identifier() string {
	return c.ppm.PublicParameters().Identifier()
}

// Auditors returns the list of auditors' identities
func (c *PublicParametersManager) Auditors() []view.Identity {
	return c.ppm.PublicParameters().Auditors()
}

// Fetch fetches the public parameters from the backend
func (c *PublicParametersManager) Fetch() ([]byte, error) {
	return c.ppm.Fetch()
}
