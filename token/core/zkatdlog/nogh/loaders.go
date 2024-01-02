/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	IsPending(id *token3.ID) (bool, error)
	GetTokenInfoAndOutputs(ids []*token3.ID, callback driver.QueryCallback2Func) error
	GetTokenOutputs(ids []*token3.ID, callback driver.QueryCallbackFunc) error
}

type VaultTokenCommitmentLoader struct {
	TokenVault TokenVault
	// Variables used to control retry condition
	NumRetries int
	RetryDelay time.Duration
}

func NewVaultTokenCommitmentLoader(tokenVault TokenVault, numRetries int, retryDelay time.Duration) *VaultTokenCommitmentLoader {
	return &VaultTokenCommitmentLoader{TokenVault: tokenVault, NumRetries: numRetries, RetryDelay: retryDelay}
}

// GetTokenOutputs takes an array of token identifiers (txID, index) and returns the corresponding token outputs
func (s *VaultTokenCommitmentLoader) GetTokenOutputs(ids []*token3.ID) ([]*token.Token, error) {
	var tokens []*token.Token

	var err error
	for i := 0; i < s.NumRetries; i++ {
		err = s.TokenVault.GetTokenOutputs(ids, func(id *token3.ID, bytes []byte) error {
			if len(bytes) == 0 {
				return errors.Errorf("failed getting serialized token output for id [%v], nil value", id)
			}
			ti := &token.Token{}
			if err := ti.Deserialize(bytes); err != nil {
				return errors.Wrapf(err, "failed deserializing token for id [%v][%s]", id, string(bytes))
			}
			tokens = append(tokens, ti)
			return nil
		})
		if err == nil {
			return tokens, nil
		}

		// check if there is any token id whose corresponding transaction is pending
		// if there is, then wait a bit and retry to load the outputs
		retry := false
		for _, id := range ids {
			pending, err := s.TokenVault.IsPending(id)
			if err != nil {
				break
			}
			if pending {
				logger.Warnf("failed getting serialized token output for id [%v] because the relative transaction is pending, retry at [%d]", id, i)
				if i >= s.NumRetries-1 {
					// too late, we tried already too many times
					return nil, errors.Wrapf(err, "failed to get token outputs, tx [%s] is still pending", id.TxId)
				}
				time.Sleep(s.RetryDelay)
				retry = true
				break
			}
		}

		if retry {
			tokens = nil
			continue
		}

		return nil, errors.Wrapf(err, "failed to get token outputs")
	}

	return nil, errors.Wrapf(err, "failed to get token outputs")
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

	// return token outputs and the corresponding opening
	if err := s.TokenVault.GetTokenInfoAndOutputs(ids, func(id *token3.ID, key string, comm, info []byte) error {
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

type PublicParamsLoader struct {
	PublicParamsFetcher driver.PublicParamsFetcher
	PPLabel             string
}

func NewPublicParamsLoader(publicParamsFetcher driver.PublicParamsFetcher, PPLabel string) *PublicParamsLoader {
	return &PublicParamsLoader{PublicParamsFetcher: publicParamsFetcher, PPLabel: PPLabel}
}

// Fetch fetches the public parameters from the backend
func (s *PublicParamsLoader) Fetch() ([]byte, error) {
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
func (s *PublicParamsLoader) FetchParams() (*crypto.PublicParams, error) {
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
