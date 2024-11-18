/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenCommitmentLoader interface {
	GetTokenOutputs(ctx context.Context, ids []*token2.ID) (map[string]*token.Token, error)
}

type TokenLoader interface {
	LoadTokens(ctx context.Context, ids []*token2.ID) ([]*token.Token, []*token.Metadata, []driver.Identity, error)
}

type Service struct {
	*common.Service[*crypto.PublicParams]
}

func NewTokenService(
	logger logging.Logger,
	ws *common.WalletService,
	ppm common.PublicParametersManager[*crypto.PublicParams],
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
	root, err := common.NewTokenService[*crypto.PublicParams](
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
	return validator.New(s.Logger, s.PublicParametersManager.PublicParams(), s.Deserializer()), nil
}
