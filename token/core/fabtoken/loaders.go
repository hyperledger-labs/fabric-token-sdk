/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
}

type VaultTokenLoader struct {
	TokenVault api2.QueryEngine
}

// GetTokens takes an array of token identifiers (txID, index) and returns the keys of the identified tokens
// in the vault and the content of the tokens
func (s *VaultTokenLoader) GetTokens(ids []*token.ID) ([]string, []*token.Token, error) {
	return s.TokenVault.GetTokens(ids...)
}

// VaultPublicParamsLoader allows one to fetch the public parameters for fabtoken
type VaultPublicParamsLoader struct {
	TokenVault          TokenVault
	PublicParamsFetcher api2.PublicParamsFetcher
	PPLabel             string
}

// Load returns the PublicParams associated with fabtoken
// Load first checks if PublicParams are cached, if not, then Load fetches them
func (s *VaultPublicParamsLoader) Load() (*PublicParams, error) {
	if s.TokenVault == nil {
		return nil, errors.New("failed to retrieve public parameters: please initialize TokenVault")
	}
	raw, err := s.TokenVault.PublicParams()
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve public parameters")
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
	pp := &PublicParams{}
	pp.Label = s.PPLabel
	err = pp.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	logger.Debugf("unmarshal public parameters done")
	return pp, nil
}

// ForceFetch returns the PublicParams associated with fabtoken
func (s *VaultPublicParamsLoader) ForceFetch() (*PublicParams, error) {
	logger.Debugf("force public parameters fetch")
	raw, err := s.PublicParamsFetcher.Fetch()
	if err != nil {
		logger.Errorf("failed retrieving public params [%s]", err)
		return nil, err
	}

	logger.Debugf("unmarshal public parameters")
	pp := &PublicParams{}
	pp.Label = s.PPLabel
	err = pp.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	logger.Debugf("unmarshal public parameters done")
	return pp, nil
}
