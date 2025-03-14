/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type Service struct {
	*common.Service[*crypto.PublicParams]
	validator *validator.Validator
}

func NewTokenService(
	logger logging.Logger,
	ws *wallet.Service,
	ppm common.PublicParametersManager[*crypto.PublicParams],
	identityProvider driver.IdentityProvider,
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

	validator, err := validator.New(logger, ppm.PublicParams(), deserializer), nil
	if err != nil {
		return nil, errors.Wrap(err, "failed to instantiate validator")
	}
	s := &Service{
		Service:   root,
		validator: validator,
	}
	return s, nil
}

func (s *Service) Validator() (driver.Validator, error) {
	return s.validator, nil
}
