/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
}

type VaultTokenLoader struct {
	TokenVault driver.QueryEngine
}

// GetTokens takes an array of token identifiers (txID, index) and returns the keys of the identified tokens
// in the vault and the content of the tokens
func (s *VaultTokenLoader) GetTokens(ids []*token.ID) ([]string, []*token.Token, error) {
	return s.TokenVault.GetTokens(ids...)
}

// PublicParamsLoader allows one to fetch the public parameters for fabtoken
type PublicParamsLoader struct {
	PublicParamsFetcher driver.PublicParamsFetcher
	PPLabel             string
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
func (s *PublicParamsLoader) FetchParams() (*PublicParams, error) {
	logger.Debugf("fetch public parameters...")
	raw, err := s.PublicParamsFetcher.Fetch()
	if err != nil {
		logger.Errorf("failed retrieving public params [%s]", err)
		return nil, err
	}

	logger.Debugf("fetched public parameters [%s], unmarshal them...", hash.Hashable(raw).String())
	pp, err := NewPublicParamsFromBytes(raw, s.PPLabel)
	if err != nil {
		return nil, err
	}
	if err := pp.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate public parameters")
	}
	logger.Debugf("fetched public parameters [%s], unmarshal them...done", hash.Hashable(raw).String())
	return pp, nil
}
