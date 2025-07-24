/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenLoader interface {
	GetTokens(ctx context.Context, ids []*token.ID) ([]*token.Token, error)
}

type Service struct {
	*common.Service[*setup.PublicParams]
}

func NewService(
	logger logging.Logger,
	ws *wallet.Service,
	ppm common.PublicParametersManager[*setup.PublicParams],
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	configuration driver.Configuration,
	issueService driver.IssueService,
	transferService driver.TransferService,
	auditorService driver.AuditorService,
	tokensService driver.TokensService,
	tokensUpgradeService driver.TokensUpgradeService,
	authorization driver.Authorization,
) (*Service, error) {
	root, err := common.NewTokenService[*setup.PublicParams](
		logger,
		ws,
		ppm,
		identityProvider,
		deserializer,
		configuration,
		nil,
		issueService,
		transferService,
		auditorService,
		tokensService,
		tokensUpgradeService,
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
	return validator.NewValidator(s.Logger, s.PublicParametersManager.PublicParams(context.Background()), s.Deserializer(), nil, nil, nil), nil
}
