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
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
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
	CM                    api3.ConfigManager

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
	cm api3.ConfigManager,
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
		CM:                    cm,
	}
	return s, nil
}

func (s *Service) DeserializeToken(tok []byte, infoRaw []byte) (*token3.Token, view.Identity, error) {
	output := &token.Token{}
	if err := output.Deserialize(tok); err != nil {
		return nil, nil, err
	}

	ti := &token.TokenInformation{}
	err := ti.Deserialize(infoRaw)
	if err != nil {
		return nil, nil, err
	}

	to, err := output.GetTokenInTheClear(ti, s.PublicParams())
	if err != nil {
		return nil, nil, err
	}

	return to, ti.Issuer, nil
}

func (s *Service) IdentityProvider() api3.IdentityProvider {
	return s.identityProvider
}

func (s *Service) Validator() api3.Validator {
	d, err := s.Deserializer()
	if err != nil {
		panic(err)
	}
	return validator.New(
		s.PublicParams(),
		d,
	)
}

func (s *Service) PublicParamsManager() api3.PublicParamsManager {
	return s.PPM
}

func (s *Service) ConfigManager() api3.ConfigManager {
	return s.CM
}

func (s *Service) PublicParams() *crypto.PublicParams {
	return s.PPM.PublicParams()
}

func (s *Service) FetchPublicParams() error {
	return s.PPM.ForceFetch()
}

func (s *Service) Deserializer() (api3.Deserializer, error) {
	d, err := s.DeserializerProvider(s.PublicParams())
	if err != nil {
		return nil, err
	}
	return d, nil
}
