/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/math"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokensService struct {
	*common.TokensService
	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
}

func NewTokensService(publicParametersManager common.PublicParametersManager[*crypto.PublicParams]) *TokensService {
	return &TokensService{TokensService: common.NewTokensService(), PublicParametersManager: publicParametersManager}
}

// DeserializeToken un-marshals a token and token info from raw bytes
// It checks if the un-marshalled token matches the token info. If not, it returns
// an error. Else it returns the token in cleartext and the identity of its issuer
func (s *TokensService) DeserializeToken(outputRaw []byte, metadataRaw []byte) (*token.Token, driver.Identity, error) {
	return s.deserializeToken(outputRaw, metadataRaw, false)
}

func (s *TokensService) IsSpendable(outputRaw []byte, metadataRaw []byte) error {
	_, _, err := s.deserializeToken(outputRaw, metadataRaw, true)
	return err
}

func (s *TokensService) deserializeToken(outputRaw []byte, metadataRaw []byte, checkOwner bool) (*token.Token, driver.Identity, error) {
	// get zkatdlog token
	output, err := s.getOutput(outputRaw, checkOwner)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get token info
	ti := &token2.Metadata{}
	err = ti.Deserialize(metadataRaw)
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

func (s *TokensService) getOutput(outputRaw []byte, checkOwner bool) (*token2.Token, error) {
	output := &token2.Token{}
	if err := output.Deserialize(outputRaw); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize oken")
	}
	if checkOwner && len(output.Owner) == 0 {
		return nil, errors.Errorf("token owner not found in output")
	}
	if err := math.CheckElement(output.Data, s.PublicParametersManager.PublicParams().Curve); err != nil {
		return nil, errors.Wrap(err, "data in invalid in output")
	}
	return output, nil
}
