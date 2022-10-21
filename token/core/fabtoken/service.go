/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type PublicParamsLoader interface {
	// Fetch fetches the public parameters from the backend
	Fetch() ([]byte, error)
	// FetchParams fetches the public parameters from the backend and unmarshal them.
	// The public parameters are also validated.
	FetchParams() (*PublicParams, error)
}

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
}

type Service struct {
	SP          view2.ServiceProvider
	TMSID       token.TMSID
	PPM         PublicParametersManager
	TokenLoader TokenLoader
	QE          QueryEngine
	CM          config.Manager

	IP                     driver.IdentityProvider
	Deserializer           driver.Deserializer
	OwnerWalletsRegistry   *identity.WalletsRegistry
	IssuerWalletsRegistry  *identity.WalletsRegistry
	AuditorWalletsRegistry *identity.WalletsRegistry
}

func NewService(
	sp view2.ServiceProvider,
	tmsID token.TMSID,
	ppm PublicParametersManager,
	tokenLoader TokenLoader,
	qe QueryEngine,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	cm config.Manager,
	kvs KVS,
) *Service {
	s := &Service{
		SP:                     sp,
		TMSID:                  tmsID,
		TokenLoader:            tokenLoader,
		QE:                     qe,
		PPM:                    ppm,
		IP:                     identityProvider,
		Deserializer:           deserializer,
		CM:                     cm,
		OwnerWalletsRegistry:   identity.NewWalletsRegistry(tmsID, identityProvider, driver.OwnerRole, kvs),
		IssuerWalletsRegistry:  identity.NewWalletsRegistry(tmsID, identityProvider, driver.IssuerRole, kvs),
		AuditorWalletsRegistry: identity.NewWalletsRegistry(tmsID, identityProvider, driver.AuditorRole, kvs),
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
