/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type PublicParameters struct {
	driver.PublicParameters
	ppm driver.PublicParamsManager
}

// Precision returns the precision used to represent the token quantity
func (c *PublicParameters) Precision() uint64 {
	return c.PublicParameters.Precision()
}

// CertificationDriver return the certification driver used to certify that tokens exist
func (c *PublicParameters) CertificationDriver() string {
	return c.PublicParameters.CertificationDriver()
}

// GraphHiding returns true if graph hiding is enabled
func (c *PublicParameters) GraphHiding() bool {
	return c.PublicParameters.GraphHiding()
}

// TokenDataHiding returns true if data hiding is enabled
func (c *PublicParameters) TokenDataHiding() bool {
	return c.PublicParameters.TokenDataHiding()
}

// MaxTokenValue returns the maximum value a token can contain
func (c *PublicParameters) MaxTokenValue() uint64 {
	return c.PublicParameters.MaxTokenValue()
}

// Serialize returns the public parameters in their serialized form
func (c *PublicParameters) Serialize() ([]byte, error) {
	return c.PublicParameters.Serialize()
}

// Identifier returns the identifier of the public parameters
func (c *PublicParameters) Identifier() string {
	return c.PublicParameters.Identifier()
}

// Auditors returns the list of auditors' identities
func (c *PublicParameters) Auditors() []Identity {
	return c.PublicParameters.Auditors()
}

// PublicParamsFetcher models the public parameters fetcher
type PublicParamsFetcher interface {
	// Fetch fetches the public parameters from the backend
	Fetch() ([]byte, error)
}

// PublicParametersManager exposes methods to manage the public parameters
type PublicParametersManager struct {
	ppm driver.PublicParamsManager
}

// PublicParameters returns the public parameters, nil if not set yet.
func (c *PublicParametersManager) PublicParameters() *PublicParameters {
	pp := c.ppm.PublicParameters()
	if pp == nil {
		return nil
	}
	return &PublicParameters{PublicParameters: pp, ppm: c.ppm}
}

func (c *PublicParametersManager) PublicParamsHash() []byte {
	return c.ppm.PublicParamsHash()
}
