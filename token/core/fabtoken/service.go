/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryEngine interface {
	IsMine(id *token2.ID) (bool, error)
	// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
	// The token type can be empty. In that case, tokens of any type are returned.
	UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error)
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

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

type Service struct {
	*WalletService
	PPM         PublicParametersManager
	TokenLoader TokenLoader
	QE          QueryEngine
	CM          config.Manager

	IP           driver.IdentityProvider
	Deserializer driver.Deserializer
}

func NewService(ws *WalletService, ppm PublicParametersManager, tokenLoader TokenLoader, qe QueryEngine, identityProvider driver.IdentityProvider, deserializer driver.Deserializer, cm config.Manager) *Service {
	s := &Service{
		WalletService: ws,
		TokenLoader:   tokenLoader,
		QE:            qe,
		PPM:           ppm,
		IP:            identityProvider,
		Deserializer:  deserializer,
		CM:            cm,
	}
	return s
}

func (s *Service) IdentityProvider() driver.IdentityProvider {
	return s.IP
}

func (s *Service) Validator() (driver.Validator, error) {
	return NewValidator(s.PPM.PublicParams(), s.Deserializer)
}

func (s *Service) PublicParamsManager() driver.PublicParamsManager {
	return s.PPM
}

func (s *Service) ConfigManager() config.Manager {
	return s.CM
}

func (s *Service) MarshalTokenRequestToSign(request *driver.TokenRequest, meta *driver.TokenRequestMetadata) ([]byte, error) {
	newReq := &driver.TokenRequest{
		Issues:    request.Issues,
		Transfers: request.Transfers,
	}
	return newReq.Bytes()
}
