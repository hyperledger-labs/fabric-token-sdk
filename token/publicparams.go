/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type PublicParamsFetcher interface {
	Fetch() ([]byte, error)
}

type PublicParametersManager struct {
	ppm tokenapi.PublicParamsManager
}

func (c *PublicParametersManager) SetAuditor(auditor []byte) ([]byte, error) {
	return c.ppm.SetAuditor(auditor)
}

func (c *PublicParametersManager) SetCertifier(certifier []byte) ([]byte, error) {
	return c.ppm.SetCertifier(certifier)
}

func (c *PublicParametersManager) AddIssuer(bytes []byte) ([]byte, error) {
	return c.ppm.AddIssuer(bytes)
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
