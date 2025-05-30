/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type PublicParametersManager[T driver.PublicParameters] interface {
	driver.PublicParamsManager
	PublicParams(ctx context.Context) T
}

type Service[T driver.PublicParameters] struct {
	Logger                  logging.Logger
	PublicParametersManager PublicParametersManager[T]
	deserializer            driver.Deserializer
	identityProvider        driver.IdentityProvider
	configuration           driver.Configuration
	certificationService    driver.CertificationService
	walletService           driver.WalletService
	issueService            driver.IssueService
	transferService         driver.TransferService
	auditorService          driver.AuditorService
	tokensService           driver.TokensService
	tokensUpgradeService    driver.TokensUpgradeService
	authorization           driver.Authorization
}

func NewTokenService[T driver.PublicParameters](
	logger logging.Logger,
	ws *wallet.Service,
	publicParametersManager PublicParametersManager[T],
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	configManager driver.Configuration,
	certificationService driver.CertificationService,
	issueService driver.IssueService,
	transferService driver.TransferService,
	auditorService driver.AuditorService,
	tokensService driver.TokensService,
	tokensUpgradeService driver.TokensUpgradeService,
	authorization driver.Authorization,
) (*Service[T], error) {
	s := &Service[T]{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		identityProvider:        identityProvider,
		deserializer:            deserializer,
		configuration:           configManager,
		certificationService:    certificationService,
		walletService:           ws,
		issueService:            issueService,
		transferService:         transferService,
		auditorService:          auditorService,
		tokensService:           tokensService,
		tokensUpgradeService:    tokensUpgradeService,
		authorization:           authorization,
	}
	return s, nil
}

// IdentityProvider returns the identity provider associated with the service
func (s *Service[T]) IdentityProvider() driver.IdentityProvider {
	return s.identityProvider
}

func (s *Service[T]) Deserializer() driver.Deserializer {
	return s.deserializer
}

func (s *Service[T]) CertificationService() driver.CertificationService {
	return s.certificationService
}

// PublicParamsManager returns the manager of the public parameters associated with the service
func (s *Service[T]) PublicParamsManager() driver.PublicParamsManager {
	return s.PublicParametersManager
}

// Configuration returns the configuration manager associated with the service
func (s *Service[T]) Configuration() driver.Configuration {
	return s.configuration
}

func (s *Service[T]) WalletService() driver.WalletService {
	return s.walletService
}

func (s *Service[T]) IssueService() driver.IssueService {
	return s.issueService
}

func (s *Service[T]) TransferService() driver.TransferService {
	return s.transferService
}

func (s *Service[T]) AuditorService() driver.AuditorService {
	return s.auditorService
}

func (s *Service[T]) TokensService() driver.TokensService {
	return s.tokensService
}

func (s *Service[T]) TokensUpgradeService() driver.TokensUpgradeService {
	return s.tokensUpgradeService
}

func (s *Service[T]) Authorization() driver.Authorization {
	return s.authorization
}

// Done releases all the resources allocated by this service
func (s *Service[T]) Done() error {
	return nil
}
