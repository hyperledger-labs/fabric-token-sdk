/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenCommitmentLoader interface {
	GetTokenOutputs(ids []*token2.ID) ([]*token.Token, error)
}

type TokenLoader interface {
	LoadTokens(ids []*token2.ID) ([]string, []*token.Token, []*token.Metadata, []driver.Identity, error)
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
	)
	if err != nil {
		return nil, err
	}

	s := &Service{
		Service: root,
	}
	return s, nil
}

// DeserializeToken un-marshals a token and token info from raw bytes
// It checks if the un-marshalled token matches the token info. If not, it returns
// an error. Else it returns the token in cleartext and the identity of its issuer
func (s *Service) DeserializeToken(tok []byte, infoRaw []byte) (*token2.Token, driver.Identity, error) {
	// get zkatdlog token
	output := &token.Token{}
	if err := output.Deserialize(tok); err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize zkatdlog token")
	}

	// get token info
	ti := &token.Metadata{}
	err := ti.Deserialize(infoRaw)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token information")
	}
	pp := s.PublicParametersManager.PublicParams()
	to, err := output.GetTokenInTheClear(ti, pp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token")
	}

	return to, ti.Issuer, nil
}

func (s *Service) GetTokenInfo(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		s.Logger.Debugf("metadata for [%s] not found", hash.Hashable(target).String())
		return nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(target).String())
	}
	return tokenInfoRaw, nil
}

func (s *Service) Validator() (driver.Validator, error) {
	return validator.New(s.Logger, s.PublicParametersManager.PublicParams(), s.Deserializer()), nil
}
