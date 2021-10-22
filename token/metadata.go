/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type queryService interface {
	// DeserializeToken returns the token and its issuer (if any).
	DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, view.Identity, error)
}

type Metadata struct {
	queryService         queryService
	tokenRequestMetadata *api2.TokenRequestMetadata
}

func (m *Metadata) GetToken(raw []byte) (*token2.Token, view.Identity, []byte, error) {
	tokenInfoRaw := m.tokenRequestMetadata.GetTokenInfo(raw)
	if len(tokenInfoRaw) == 0 {
		logger.Debugf("metadata for [%s] not found", hash.Hashable(raw).String())
		return nil, nil, nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(raw).String())
	}
	tok, id, err := m.queryService.DeserializeToken(raw, tokenInfoRaw)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed getting token in the clear")
	}
	return tok, id, tokenInfoRaw, nil
}

func (m *Metadata) SpentTokenID() []*token2.ID {
	var res []*token2.ID
	for _, transfer := range m.tokenRequestMetadata.Transfers {
		res = append(res, transfer.TokenIDs...)
	}
	return res
}
