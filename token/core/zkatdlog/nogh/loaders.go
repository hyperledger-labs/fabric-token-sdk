/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
	GetTokenInfoAndCommitments(ids []*token3.ID, callback driver.QueryCallback2Func) error
	GetTokenCommitments(ids []*token3.ID, callback driver.QueryCallbackFunc) error
}

type VaultTokenCommitmentLoader struct {
	TokenVault TokenVault
}

// GetTokenCommitments takes an array of token identifiers (txID, index) and returns the corresponding tokens
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

// LoadTokens takes an array of token identifiers (txID, index) and returns the keys in the vault
// matching the token identifiers, the corresponding zkatdlog tokens, the information of the
// tokens in clear text and the identities of their owners
// LoadToken returns an error in case of failure
func (s *VaultTokenLoader) LoadTokens(ids []*token3.ID) ([]string, []*token.Token, []*token.Metadata, []view.Identity, error) {
	var tokens []*token.Token
	var inputIDs []string
	var inputInf []*token.Metadata
	var signerIds []view.Identity

	// return token commitments and the corresponding opening
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
		ti := &token.Metadata{}
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
	PublicParamsFetcher driver.PublicParamsFetcher
	PPLabel             string
}

func NewVaultPublicParamsLoader(publicParamsFetcher driver.PublicParamsFetcher, PPLabel string) *VaultPublicParamsLoader {
	return &VaultPublicParamsLoader{PublicParamsFetcher: publicParamsFetcher, PPLabel: PPLabel}
}

// Fetch fetches the public parameters from the backend
func (s *VaultPublicParamsLoader) Fetch() ([]byte, error) {
	logger.Debugf("fetch public parameters...")
	raw, err := s.PublicParamsFetcher.Fetch()
	if err != nil {
		logger.Errorf("failed retrieving public params [%s]", err)
		return nil, err
	}
	logger.Debugf("fetched public parameters [%s]", hash.Hashable(raw).String())
	return raw, nil
}

// FetchParams fetches the public parameters from the backend and unmarshal them
func (s *VaultPublicParamsLoader) FetchParams() (*crypto.PublicParams, error) {
	logger.Debugf("fetch public parameters...")
	raw, err := s.PublicParamsFetcher.Fetch()
	if err != nil {
		logger.Errorf("failed retrieving public params [%s]", err)
		return nil, err
	}

	logger.Debugf("fetched public parameters [%s], unmarshal them...", hash.Hashable(raw).String())
	pp := &crypto.PublicParams{}
	pp.Label = s.PPLabel
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal public parameters")
	}
	logger.Debugf("fetched public parameters [%s], unmarshal them...done", hash.Hashable(raw).String())
	if err := pp.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate public parameters")
	}
	return pp, nil
}
