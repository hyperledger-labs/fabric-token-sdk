/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenLoader interface {
	GetTokens(ids []*token.ID) ([]*token.Token, error)
}

type Service struct {
	*common.Service[*PublicParams]
}

func NewService(
	logger logging.Logger,
	ws *wallet.Service,
	ppm common.PublicParametersManager[*PublicParams],
	identityProvider driver.IdentityProvider,
	serializer driver.Serializer,
	deserializer driver.Deserializer,
	configuration driver.Configuration,
	issueService driver.IssueService,
	transferService driver.TransferService,
	auditorService driver.AuditorService,
	tokensService driver.TokensService,
	authorization driver.Authorization,
) (*Service, error) {
	root, err := common.NewTokenService[*PublicParams](
		logger,
		ws,
		ppm,
		identityProvider,
		serializer,
		deserializer,
		configuration,
		nil,
		issueService,
		transferService,
		auditorService,
		tokensService,
		authorization,
	)
	if err != nil {
		return nil, err
	}

	s := &Service{
		Service: root,
	}
	return s, nil
}

func (s *Service) Validator() (driver.Validator, error) {
	return NewValidator(s.Logger, s.PublicParametersManager.PublicParams(), s.Deserializer()), nil
}
