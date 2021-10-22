/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
}

type VaultTokenLoader struct {
	TokenVault api2.QueryEngine
}

func (s *VaultTokenLoader) GetTokens(ids []*token.ID) ([]string, []*token.Token, error) {
	return s.TokenVault.GetTokens(ids...)
}

type VaultPublicParamsLoader struct {
	TokenVault          TokenVault
	PublicParamsFetcher api2.PublicParamsFetcher
	PPLabel             string
}

func (s *VaultPublicParamsLoader) Load() (*PublicParams, error) {
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
	pp := &PublicParams{}
	pp.Label = s.PPLabel
	err = pp.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	logger.Debugf("unmarshal public parameters done")
	return pp, nil
}

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
