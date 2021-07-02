/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Channel interface {
	Name() string
	Vault() *fabric.Vault
}

type PublicParamsLoader interface {
	Load() (*PublicParams, error)
	ForceFetch() (*PublicParams, error)
}

type QueryEngine interface {
	IsMine(id *token2.Id) (bool, error)
	ListUnspentTokens() (*token2.UnspentTokens, error)
	ListAuditTokens(ids ...*token2.Id) ([]*token2.Token, error)
	ListHistoryIssuedTokens() (*token2.IssuedTokens, error)
	PublicParams() ([]byte, error)
}

type TokenLoader interface {
	GetTokens(ids []*token2.Id) ([]string, []*token2.Token, error)
}

type PublicParametersManager interface {
	driver.PublicParamsManager
	AuditorIdentity() view.Identity
}

type service struct {
	sp          view2.ServiceProvider
	channel     Channel
	namespace   string
	ppm         PublicParametersManager
	tokenLoader TokenLoader
	qe          QueryEngine

	identityProvider driver.IdentityProvider
	deserializer     driver.Deserializer
	ownerWallets     []*ownerWallet
	issuerWallets    []*issuerWallet
	auditorWallets   []*auditorWallet
	walletsLock      sync.Mutex
}

func NewService(
	sp view2.ServiceProvider,
	channel Channel,
	namespace string,
	ppm PublicParametersManager,
	tokenLoader TokenLoader,
	qe QueryEngine,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
) *service {
	s := &service{
		sp:               sp,
		channel:          channel,
		namespace:        namespace,
		tokenLoader:      tokenLoader,
		qe:               qe,
		ppm:              ppm,
		identityProvider: identityProvider,
		deserializer:     deserializer,
	}
	return s
}

func (s *service) IdentityProvider() driver.IdentityProvider {
	return s.identityProvider
}

func (s *service) Validator() driver.Validator {
	return NewValidator(s.ppm, s.deserializer)
}

func (s *service) PublicParamsManager() driver.PublicParamsManager {
	return s.ppm
}
