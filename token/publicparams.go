/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type PublicParamsFetcher interface {
	Fetch() ([]byte, error)
}

// PublicParametersManager exposes methods to manage the public parameters
// TODO: add precision
type PublicParametersManager struct {
	ppm tokenapi.PublicParamsManager
}

func (c *PublicParametersManager) CertificationDriver() string {
	return c.ppm.PublicParameters().CertificationDriver()
}

func (c *PublicParametersManager) GraphHiding() bool {
	return c.ppm.PublicParameters().GraphHiding()
}

func (c *PublicParametersManager) TokenDataHiding() bool {
	return c.ppm.PublicParameters().TokenDataHiding()
}

func (c *PublicParametersManager) MaxTokenValue() uint64 {
	return c.ppm.PublicParameters().MaxTokenValue()
}

func (c *PublicParametersManager) Bytes() ([]byte, error) {
	return c.ppm.PublicParameters().Bytes()
}

func (c *PublicParametersManager) ForceFetch() error {
	return c.ppm.ForceFetch()
}

func (c *PublicParametersManager) Identifier() string {
	return c.ppm.PublicParameters().Identifier()
}
