/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/api"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
}

type VaultPublicParamsLoader struct {
	TokenVault          TokenVault
	PublicParamsFetcher api2.PublicParamsFetcher
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
	err = pp.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	logger.Debugf("unmarshal public parameters done")
	return pp, nil
}

func (s *VaultPublicParamsLoader) SetPublicParamsFetcher(f api2.PublicParamsFetcher) {
	s.PublicParamsFetcher = f
}
