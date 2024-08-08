/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type PublicParametersManager[T driver.PublicParameters] interface {
	driver.PublicParamsManager
	PublicParams() T
}

type Service[T driver.PublicParameters] struct {
	Logger                  logging.Logger
	PublicParametersManager PublicParametersManager[T]
	serializer              driver.Serializer
	deserializer            driver.Deserializer
	identityProvider        driver.IdentityProvider
	configuration           driver.Configuration
	certificationService    driver.CertificationService
	walletService           driver.WalletService
	issueService            driver.IssueService
	transferService         driver.TransferService
	auditorService          driver.AuditorService
	tokensService           driver.TokensService
	authorization           driver.Authorization
}

func NewTokenService[T driver.PublicParameters](
	logger logging.Logger,
	ws *WalletService,
	publicParametersManager PublicParametersManager[T],
	identityProvider driver.IdentityProvider,
	serializer driver.Serializer,
	deserializer driver.Deserializer,
	configManager driver.Configuration,
	certificationService driver.CertificationService,
	issueService driver.IssueService,
	transferService driver.TransferService,
	auditorService driver.AuditorService,
	tokensService driver.TokensService,
	authorization driver.Authorization,
) (*Service[T], error) {
	s := &Service[T]{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		identityProvider:        identityProvider,
		serializer:              serializer,
		deserializer:            deserializer,
		configuration:           configManager,
		certificationService:    certificationService,
		walletService:           ws,
		issueService:            issueService,
		transferService:         transferService,
		auditorService:          auditorService,
		tokensService:           tokensService,
		authorization:           authorization,
	}
	return s, nil
}

func (s *Service[T]) GetTokenInfo(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		s.Logger.Debugf("metadata for [%s] not found", Hashable(target))
		return nil, errors.Errorf("metadata for [%s] not found", Hashable(target))
	}
	return tokenInfoRaw, nil
}

// IdentityProvider returns the identity provider associated with the service
func (s *Service[T]) IdentityProvider() driver.IdentityProvider {
	return s.identityProvider
}

func (s *Service[T]) Deserializer() driver.Deserializer {
	return s.deserializer
}

func (s *Service[T]) Serializer() driver.Serializer {
	return s.serializer
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

func (s *Service[T]) Authorization() driver.Authorization {
	return s.authorization
}

// Done releases all the resources allocated by this service
func (s *Service[T]) Done() error {
	return nil
}
