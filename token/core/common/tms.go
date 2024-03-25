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
	*WalletService
	Serializer
	Logger                  *flogging.FabricLogger
	PublicParametersManager PublicParametersManager[T]
	deserializer            driver.Deserializer
	identityProvider        driver.IdentityProvider
	configManager           config.Manager
	certificationService    driver.CertificationService
}

func NewTokenService[T driver.PublicParameters](
	logger *flogging.FabricLogger,
	ws *WalletService,
	publicParametersManager PublicParametersManager[T],
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	configManager config.Manager,
	certificationService driver.CertificationService,
) (*Service[T], error) {
	s := &Service[T]{
		Logger:                  logger,
		WalletService:           ws,
		PublicParametersManager: publicParametersManager,
		identityProvider:        identityProvider,
		deserializer:            deserializer,
		configManager:           configManager,
		certificationService:    certificationService,
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

func (s *Service[T]) Done() error {
	return nil
}
