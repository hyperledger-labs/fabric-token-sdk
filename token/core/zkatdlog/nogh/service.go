/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
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

type QueryEngine interface {
	// IsPending returns true if the transaction the passed id refers to is still pending, false otherwise
	IsPending(id *token3.ID) (bool, error)
	IsMine(id *token3.ID) (bool, error)
	// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
	// The token type can be empty. In that case, tokens of any type are returned.
	UnspentTokensIteratorBy(id, tokenType string) (driver.UnspentTokensIterator, error)
	ListAuditTokens(ids ...*token3.ID) ([]*token3.Token, error)
	ListHistoryIssuedTokens() (*token3.IssuedTokens, error)
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
	SP                    view2.ServiceProvider
	TMSID                 token2.TMSID
	PP                    *crypto.PublicParams
	PPM                   PublicParametersManager
	PPLabel               string
	PublicParamsFetcher   driver.PublicParamsFetcher
	TokenLoader           TokenLoader
	TokenCommitmentLoader TokenCommitmentLoader
	QE                    QueryEngine
	DeserializerProvider  DeserializerProviderFunc
	configManager         config.Manager

	identityProvider       driver.IdentityProvider
	OwnerWalletsRegistry   *identity.WalletsRegistry
	IssuerWalletsRegistry  *identity.WalletsRegistry
	AuditorWalletsRegistry *identity.WalletsRegistry
}

func NewTokenService(
	sp view2.ServiceProvider,
	tmsID token2.TMSID,
	PPM PublicParametersManager,
	tokenLoader TokenLoader,
	tokenCommitmentLoader TokenCommitmentLoader,
	queryEngine QueryEngine,
	identityProvider driver.IdentityProvider,
	deserializerProvider DeserializerProviderFunc,
	ppLabel string,
	configManager config.Manager,
	kvs KVS,
) (*Service, error) {
	s := &Service{
		TMSID:                  tmsID,
		SP:                     sp,
		PPM:                    PPM,
		TokenLoader:            tokenLoader,
		TokenCommitmentLoader:  tokenCommitmentLoader,
		QE:                     queryEngine,
		identityProvider:       identityProvider,
		DeserializerProvider:   deserializerProvider,
		PPLabel:                ppLabel,
		configManager:          configManager,
		OwnerWalletsRegistry:   identity.NewWalletsRegistry(tmsID, identityProvider, driver.OwnerRole, kvs),
		IssuerWalletsRegistry:  identity.NewWalletsRegistry(tmsID, identityProvider, driver.IssuerRole, kvs),
		AuditorWalletsRegistry: identity.NewWalletsRegistry(tmsID, identityProvider, driver.AuditorRole, kvs),
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

// PublicParams returns the public parameters associated with the service
func (s *Service) PublicParams() *crypto.PublicParams {
	return s.PPM.PublicParams()
}

func (s *Service) NewRequest() driver.TokenRequest {
	return &common.TokenRequest{}
}

func (s *Service) NewRequestMetadata() *driver.TokenRequestMetadata {
	return &driver.TokenRequestMetadata{}
}

func (s *Service) LoadPublicParams() error {
	return s.PPM.Load()
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

func (s *Service) MarshalTokenRequestToSign(request driver.TokenRequest, meta *driver.TokenRequestMetadata) ([]byte, error) {
	req, ok := request.(*common.TokenRequest)
	if !ok {
		return nil, errors.Errorf("expect *common.TokenRequest, got [%T]", request)
	}
	return req.MarshalTokenRequestToSign(meta)
}

func (s *Service) MarshalToAudit(anchor string, request driver.TokenRequest, metadata *driver.TokenRequestMetadata) ([]byte, error) {
	req, ok := request.(*common.TokenRequest)
	if !ok {
		return nil, errors.Errorf("expect *common.TokenRequest, got [%T]", request)
	}
	return req.MarshalToAudit(anchor, metadata)
}
