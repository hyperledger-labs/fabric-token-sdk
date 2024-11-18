/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

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
