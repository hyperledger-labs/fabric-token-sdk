/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
	GetTokenInfoAndCommitments(ids []*token3.ID, callback api2.QueryCallback2Func) error
	GetTokenCommitments(ids []*token3.ID, callback api2.QueryCallbackFunc) error
}

type VaultTokenCommitmentLoader struct {
	TokenVault TokenVault
}

func (s *VaultTokenCommitmentLoader) GetTokenCommitments(ids []*token3.ID) ([]*token.Token, error) {
	var tokens []*token.Token
	if err := s.TokenVault.GetTokenCommitments(ids, func(id *token3.ID, bytes []byte) error {
		if len(bytes) == 0 {
			return errors.Errorf("failed getting state for id [%v], nil value", id)
		}
		ti := &token.Token{}
		if err := ti.Deserialize(bytes); err != nil {
			return errors.Wrapf(err, "failed deserializeing token for id [%v][%s]", id, string(bytes))
		}
		tokens = append(tokens, ti)
		return nil
	}); err != nil {
		return nil, err
	}
	return tokens, nil
}

type VaultTokenLoader struct {
	TokenVault TokenVault
}

func (s *VaultTokenLoader) LoadTokens(ids []*token3.ID) ([]string, []*token.Token, []*token.TokenInformation, []view.Identity, error) {
	var tokens []*token.Token
	var inputIDs []string
	var inputInf []*token.TokenInformation
	var signerIds []view.Identity

	if err := s.TokenVault.GetTokenInfoAndCommitments(ids, func(id *token3.ID, key string, comm, info []byte) error {
		if len(comm) == 0 {
			return errors.Errorf("failed getting state for id [%v], nil comm value", id)
		}
		if len(info) == 0 {
			return errors.Errorf("failed getting state for id [%v], nil info value", id)
		}

		logger.Debugf("loaded transfer input [%s]", hash.Hashable(comm).String())
		tok := &token.Token{}
		err := tok.Deserialize(comm)
		if err != nil {
			return errors.Wrapf(err, "failed unmarshalling token for id [%v]", id)
		}
		ti := &token.TokenInformation{}
		err = ti.Deserialize(info)
		if err != nil {
			return errors.Wrapf(err, "failed deserializeing token info for id [%v]", id)
		}

		inputIDs = append(inputIDs, key)
		tokens = append(tokens, tok)
		inputInf = append(inputInf, ti)
		signerIds = append(signerIds, tok.Owner)

		return nil
	}); err != nil {
		return nil, nil, nil, nil, err
	}

	return inputIDs, tokens, inputInf, signerIds, nil
}

type VaultPublicParamsLoader struct {
	TokenVault          TokenVault
	PublicParamsFetcher api2.PublicParamsFetcher
	PPLabel             string
}

func (s *VaultPublicParamsLoader) Load() (*crypto.PublicParams, error) {
	raw, err := s.TokenVault.PublicParams()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		logger.Warnf("public parameters not found")
		raw, err = s.PublicParamsFetcher.Fetch()
		if err != nil {
			logger.Errorf("failed retrieving public params [%s]", err)
			return nil, err
		}
	}

	logger.Debugf("unmarshal public parameters")
	pp := &crypto.PublicParams{}
	pp.Label = s.PPLabel
	err = pp.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	logger.Debugf("unmarshal public parameters done")
	return pp, nil
}

func (s *VaultPublicParamsLoader) ForceFetch() (*crypto.PublicParams, error) {
	logger.Debugf("force public parameters fetch")
	raw, err := s.PublicParamsFetcher.Fetch()
	if err != nil {
		logger.Errorf("failed retrieving public params [%s]", err)
		return nil, err
	}

	logger.Debugf("unmarshal public parameters")
	pp := &crypto.PublicParams{}
	pp.Label = s.PPLabel
	err = pp.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	logger.Debugf("unmarshal public parameters done")
	return pp, nil
}
