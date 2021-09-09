/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	"reflect"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

type TokenManagerService interface {
	IssueService
	TransferService
	TokenService
	AuditorService
	WalletService
	CertificationService
	Deserializer

	Validator() Validator
	PublicParamsManager() PublicParamsManager
}

type TokenManagerServiceProvider interface {
	// GetTokenManagerService returns a TokenManagerService instance for the passed parameters
	// If a TokenManagerService is not available, it creates one.
	GetTokenManagerService(network string, channel string, namespace string, publicParamsFetcher PublicParamsFetcher) (TokenManagerService, error)
}

func GetTokenManagementService(ctx view2.ServiceProvider) TokenManagerServiceProvider {
	s, err := ctx.GetService(reflect.TypeOf((*TokenManagerServiceProvider)(nil)))
	if err != nil {
		panic(err)
	}
	return s.(TokenManagerServiceProvider)
}
