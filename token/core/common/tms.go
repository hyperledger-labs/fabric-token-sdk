/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

type PublicParametersManager[T driver.PublicParameters] interface {
	driver.PublicParamsManager
	PublicParams() T
}

type Service[T driver.PublicParameters] struct {
	Logger                  *flogging.FabricLogger
	PublicParametersManager PublicParametersManager[T]
	serializer              driver.Serializer
	deserializer            driver.Deserializer
	identityProvider        driver.IdentityProvider
	configManager           config.Manager
	certificationService    driver.CertificationService
	walletService           driver.WalletService
	issueService            driver.IssueService
	transferService         driver.TransferService
	auditorService          driver.AuditorService
	tokensService           driver.TokensService
}

func NewTokenService[T driver.PublicParameters](
	logger *flogging.FabricLogger,
	ws *WalletService,
	publicParametersManager PublicParametersManager[T],
	identityProvider driver.IdentityProvider,
	serializer driver.Serializer,
	deserializer driver.Deserializer,
	configManager config.Manager,
	certificationService driver.CertificationService,
	issueService driver.IssueService,
	transferService driver.TransferService,
	auditorService driver.AuditorService,
	tokensService driver.TokensService,
) (*Service[T], error) {
	s := &Service[T]{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		identityProvider:        identityProvider,
		serializer:              serializer,
		deserializer:            deserializer,
		configManager:           configManager,
		certificationService:    certificationService,
		walletService:           ws,
		issueService:            issueService,
		transferService:         transferService,
		auditorService:          auditorService,
		tokensService:           tokensService,
	}
	return s, nil
}

func (s *Service[T]) GetTokenInfo(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		s.Logger.Debugf("metadata for [%s] not found", hash.Hashable(target).String())
		return nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(target).String())
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

// ConfigManager returns the configuration manager associated with the service
func (s *Service[T]) ConfigManager() config.Manager {
	return s.configManager
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

// Done releases all the resources allocated by this service
func (s *Service[T]) Done() error {
	return nil
}
