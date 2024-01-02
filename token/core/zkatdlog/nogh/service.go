/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenCommitmentLoader interface {
	GetTokenOutputs(ids []*token3.ID) ([]*token.Token, error)
}

type TokenLoader interface {
	LoadTokens(ids []*token3.ID) ([]string, []*token.Token, []*token.Metadata, []view.Identity, error)
}

type PublicParametersManager interface {
	driver.PublicParamsManager
	PublicParams() *crypto.PublicParams
}

type DeserializerProviderFunc = func(params *crypto.PublicParams) (driver.Deserializer, error)

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

type Service struct {
	*WalletService
	PPM                   PublicParametersManager
	TokenLoader           TokenLoader
	TokenCommitmentLoader TokenCommitmentLoader
	DeserializerProvider  DeserializerProviderFunc
	identityProvider      driver.IdentityProvider
	configManager         config.Manager
}

func NewTokenService(ws *WalletService, PPM PublicParametersManager, tokenLoader TokenLoader, tokenCommitmentLoader TokenCommitmentLoader, identityProvider driver.IdentityProvider, deserializerProvider DeserializerProviderFunc, configManager config.Manager) (*Service, error) {
	s := &Service{
		WalletService:         ws,
		PPM:                   PPM,
		TokenLoader:           tokenLoader,
		TokenCommitmentLoader: tokenCommitmentLoader,
		identityProvider:      identityProvider,
		DeserializerProvider:  deserializerProvider,
		configManager:         configManager,
	}
	return s, nil
}

// DeserializeToken un-marshals a token and token info from raw bytes
// It checks if the un-marshalled token matches the token info. If not, it returns
// an error. Else it returns the token in cleartext and the identity of its issuer
func (s *Service) DeserializeToken(tok []byte, infoRaw []byte) (*token3.Token, view.Identity, error) {
	// get zkatdlog token
	output := &token.Token{}
	if err := output.Deserialize(tok); err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize zkatdlog token")
	}

	// get token info
	ti := &token.Metadata{}
	err := ti.Deserialize(infoRaw)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token information")
	}
	pp := s.PublicParams()
	if pp == nil {
		return nil, nil, errors.Errorf("public parameters not inizialized")
	}
	to, err := output.GetTokenInTheClear(ti, pp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token")
	}

	return to, ti.Issuer, nil
}

func (s *Service) GetTokenInfo(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		logger.Debugf("metadata for [%s] not found", hash.Hashable(target).String())
		return nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(target).String())
	}
	return tokenInfoRaw, nil
}

// IdentityProvider returns the identity provider associated with the service
func (s *Service) IdentityProvider() driver.IdentityProvider {
	return s.identityProvider
}

// Validator returns the validator associated with the service
func (s *Service) Validator() (driver.Validator, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get deserializer")
	}
	pp := s.PublicParams()
	if pp == nil {
		return nil, errors.Errorf("public parameters not inizialized")
	}
	return validator.New(pp, d), nil
}

// PublicParamsManager returns the manager of the public parameters associated with the service
func (s *Service) PublicParamsManager() driver.PublicParamsManager {
	return s.PPM
}

// ConfigManager returns the configuration manager associated with the service
func (s *Service) ConfigManager() config.Manager {
	return s.configManager
}

func (s *Service) Deserializer() (driver.Deserializer, error) {
	pp := s.PublicParams()
	if pp == nil {
		return nil, errors.Errorf("public parameters not inizialized")
	}
	d, err := s.DeserializerProvider(pp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d, nil
}

func (s *Service) MarshalTokenRequestToSign(request *driver.TokenRequest, meta *driver.TokenRequestMetadata) ([]byte, error) {
	newReq := &driver.TokenRequest{
		Issues:    request.Issues,
		Transfers: request.Transfers,
	}
	return newReq.Bytes()
}

// PublicParams returns the public parameters associated with the service
func (s *Service) PublicParams() *crypto.PublicParams {
	return s.PPM.PublicParams()
}
