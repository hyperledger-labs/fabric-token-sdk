/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type TokenManagerService interface {
	IssueService
	TransferService
	TokenService
	AuditorService
	WalletService
	CertificationService
	Deserializer

	IdentityProvider() IdentityProvider
	Validator() Validator
	PublicParamsManager() PublicParamsManager
	ConfigManager() ConfigManager
}

type TokenManagerServiceProvider interface {
	// GetTokenManagerService returns a TokenManagerService instance for the passed parameters
	// If a TokenManagerService is not available, it creates one.
	GetTokenManagerService(network string, channel string, namespace string, publicParamsFetcher PublicParamsFetcher) (TokenManagerService, error)
}
