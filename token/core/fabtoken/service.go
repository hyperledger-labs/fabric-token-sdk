/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenLoader interface {
	GetTokens(ids []*token.ID) ([]string, []*token.Token, error)
}

type Service struct {
	*common.Service[*PublicParams]
	TokenLoader TokenLoader
}

func NewService(
	ws *common.WalletService,
	ppm common.PublicParametersManager[*PublicParams],
	tokenLoader TokenLoader,
	identityProvider driver.IdentityProvider,
	serializer driver.Serializer,
	deserializer driver.Deserializer,
	configManager config.Manager,
	issueService driver.IssueService,
) (*Service, error) {
	root, err := common.NewTokenService[*PublicParams](
		logger,
		ws,
		ppm,
		identityProvider,
		serializer,
		deserializer,
		configManager,
		nil,
		issueService,
	)
	if err != nil {
		return nil, err
	}

	s := &Service{
		Service:     root,
		TokenLoader: tokenLoader,
	}
	return s, nil
}

func (s *Service) Validator() (driver.Validator, error) {
	return NewValidator(s.PublicParametersManager.PublicParams(), s.Deserializer()), nil
}
