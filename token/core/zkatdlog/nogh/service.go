/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenCommitmentLoader interface {
	GetTokenCommitments(ids []*token3.ID) ([]*token.Token, error)
}

type QueryEngine interface {
	IsMine(id *token3.ID) (bool, error)
	ListUnspentTokens() (*token3.UnspentTokens, error)
	ListAuditTokens(ids ...*token3.ID) ([]*token3.Token, error)
	ListHistoryIssuedTokens() (*token3.IssuedTokens, error)
}

type TokenLoader interface {
	LoadTokens(ids []*token3.ID) ([]string, []*token.Token, []*token.TokenInformation, []view.Identity, error)
}

type PublicParametersManager interface {
	api3.PublicParamsManager
	PublicParams() *crypto.PublicParams
}

type DeserializerProviderFunc = func(params *crypto.PublicParams) (api3.Deserializer, error)

type Service struct {
	Channel               string
	Namespace             string
	SP                    view2.ServiceProvider
	PP                    *crypto.PublicParams
	PPM                   PublicParametersManager
	PPLabel               string
	PublicParamsFetcher   api3.PublicParamsFetcher
	TokenLoader           TokenLoader
	TokenCommitmentLoader TokenCommitmentLoader
	QE                    QueryEngine
	DeserializerProvider  DeserializerProviderFunc
	configManager         config.Manager

	identityProvider api3.IdentityProvider
	OwnerWallets     []*wallet
	IssuerWallets    []*issuerWallet
	AuditorWallets   []*auditorWallet
	WalletsLock      sync.Mutex
}

func NewTokenService(
	channel string,
	namespace string,
	sp view2.ServiceProvider,
	PPM PublicParametersManager,
	tokenLoader TokenLoader,
	tokenCommitmentLoader TokenCommitmentLoader,
	queryEngine QueryEngine,
	identityProvider api3.IdentityProvider,
	deserializerProvider DeserializerProviderFunc,
	ppLabel string,
	configManager config.Manager,
) (*Service, error) {
	s := &Service{
		Channel:               channel,
		Namespace:             namespace,
		SP:                    sp,
		PPM:                   PPM,
		TokenLoader:           tokenLoader,
		TokenCommitmentLoader: tokenCommitmentLoader,
		QE:                    queryEngine,
		identityProvider:      identityProvider,
		DeserializerProvider:  deserializerProvider,
		PPLabel:               ppLabel,
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
	ti := &token.TokenInformation{}
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
func (s *Service) IdentityProvider() api3.IdentityProvider {
	return s.identityProvider
}

// Validator returns the validator associated with the service
func (s *Service) Validator() api3.Validator {
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
func (s *Service) PublicParamsManager() api3.PublicParamsManager {
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
	return s.PPM.ForceFetch()
}

func (s *Service) Deserializer() (api3.Deserializer, error) {
	pp := s.PublicParams()
	d, err := s.DeserializerProvider(pp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d, nil
}
