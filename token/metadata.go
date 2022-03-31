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

// Metadata contains the metadata of a Token Request
type Metadata struct {
	queryService         queryService
	tokenRequestMetadata *api2.TokenRequestMetadata
}

// GetToken unmarshals the given bytes to extract the token and its issuer (if any).
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

// SpentTokenID returns the token IDs of the tokens that were spent by the Token Request this metadata is associated with.
func (m *Metadata) SpentTokenID() []*token2.ID {
	var res []*token2.ID
	for _, transfer := range m.tokenRequestMetadata.Transfers {
		res = append(res, transfer.TokenIDs...)
	}
	return res
}
