/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
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
	GetTokenCommitments(ids []*token3.ID) ([]*token.Token, error)
}

type QueryEngine interface {
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
	to, err := output.GetTokenInTheClear(ti, pp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token")
	}

	return to, ti.Issuer, nil
}

// IdentityProvider returns the identity provider associated with the service
func (s *Service) IdentityProvider() driver.IdentityProvider {
	return s.identityProvider
}

// Validator returns the validator associated with the service
func (s *Service) Validator() driver.Validator {
	d, err := s.Deserializer()
	if err != nil {
		panic(err)
	}
	pp := s.PublicParams()
	return validator.New(
		pp,
		d,
	)
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

func (s *Service) FetchPublicParams() error {
	return s.PPM.Update()
}

func (s *Service) Deserializer() (driver.Deserializer, error) {
	pp := s.PublicParams()
	d, err := s.DeserializerProvider(pp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d, nil
}
