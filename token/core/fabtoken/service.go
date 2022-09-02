/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"fmt"
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type PublicParamsLoader interface {
	// Fetch fetches the public parameters from the backend
	Fetch() ([]byte, error)
	// FetchParams fetches the public parameters from the backend and unmarshal them
	FetchParams() (*PublicParams, error)
}

type QueryEngine interface {
	IsMine(id *token2.ID) (bool, error)
	ListUnspentTokens() (*token2.UnspentTokens, error)
	ListAuditTokens(ids ...*token2.ID) ([]*token2.Token, error)
	ListHistoryIssuedTokens() (*token2.IssuedTokens, error)
	PublicParams() ([]byte, error)
}

type TokenLoader interface {
	GetTokens(ids []*token2.ID) ([]string, []*token2.Token, error)
}

type PublicParametersManager interface {
	driver.PublicParamsManager
	PublicParams() *PublicParams
}

type Service struct {
	SP          view2.ServiceProvider
	Channel     string
	Namespace   string
	PPM         PublicParametersManager
	TokenLoader TokenLoader
	QE          QueryEngine
	CM          config.Manager

	IP             driver.IdentityProvider
	Deserializer   driver.Deserializer
	OwnerWallets   []*ownerWallet
	IssuerWallets  []*issuerWallet
	AuditorWallets []*auditorWallet
	WalletsLock    sync.Mutex
}

func NewService(
	sp view2.ServiceProvider,
	channel string,
	namespace string,
	ppm PublicParametersManager,
	tokenLoader TokenLoader,
	qe QueryEngine,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	cm config.Manager,
) *Service {
	s := &Service{
		SP:           sp,
		Namespace:    namespace,
		Channel:      channel,
		TokenLoader:  tokenLoader,
		QE:           qe,
		PPM:          ppm,
		IP:           identityProvider,
		Deserializer: deserializer,
		CM:           cm,
	}
	return s
}

func (s *Service) IdentityProvider() driver.IdentityProvider {
	return s.IP
}

func (s *Service) Validator() driver.Validator {
	v, err := NewValidator(s.PPM.PublicParams(), s.Deserializer)
	if err != nil {
		panic(fmt.Sprintf("failed to create validator: %s", err))
	}
	return v
}

func (s *Service) PublicParamsManager() driver.PublicParamsManager {
	return s.PPM
}

func (s *Service) ConfigManager() config.Manager {
	return s.CM
}
