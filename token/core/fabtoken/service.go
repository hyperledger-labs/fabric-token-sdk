/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type PublicParamsLoader interface {
	Load() (*PublicParams, error)
	ForceFetch() (*PublicParams, error)
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
	AuditorIdentity() view.Identity
}

type Service struct {
	SP          view2.ServiceProvider
	Namespace   string
	PPM         PublicParametersManager
	TokenLoader TokenLoader
	QE          QueryEngine

	IP             driver.IdentityProvider
	Deserializer   driver.Deserializer
	OwnerWallets   []*ownerWallet
	IssuerWallets  []*issuerWallet
	AuditorWallets []*auditorWallet
	WalletsLock    sync.Mutex
}

func NewService(
	sp view2.ServiceProvider,
	namespace string,
	ppm PublicParametersManager,
	tokenLoader TokenLoader,
	qe QueryEngine,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
) *Service {
	s := &Service{
		SP:           sp,
		Namespace:    namespace,
		TokenLoader:  tokenLoader,
		QE:           qe,
		PPM:          ppm,
		IP:           identityProvider,
		Deserializer: deserializer,
	}
	return s
}

func (s *Service) IdentityProvider() driver.IdentityProvider {
	return s.IP
}

func (s *Service) Validator() driver.Validator {
	return NewValidator(s.PPM, s.Deserializer)
}

func (s *Service) PublicParamsManager() driver.PublicParamsManager {
	return s.PPM
}
