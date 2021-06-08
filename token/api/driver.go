/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package api

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

type Channel interface {
	Name() string
	Vault() *fabric.Vault
}

type Driver interface {
	PublicParametersFromBytes(params []byte) (PublicParameters, error)
	NewTokenService(sp view2.ServiceProvider, publicParamsFetcher PublicParamsFetcher, network string, channel Channel, namespace string) (TokenManagerService, error)
	NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error)
	NewValidator(pp PublicParameters) (Validator, error)
}
