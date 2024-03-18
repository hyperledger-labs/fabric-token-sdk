/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// DeserializeToken returns a deserialized token and the identity of its issuer
func (s *Service) DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, view.Identity, error) {
	tok := &token2.Token{}
	if err := json.Unmarshal(outputRaw, tok); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token")
	}

	tokInfo := &OutputMetadata{}
	if err := tokInfo.Deserialize(tokenInfoRaw); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token information")
	}

	return tok, tokInfo.Issuer, nil
}

func (s *Service) GetTokenInfo(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		logger.Debugf("metadata for [%s] not found", hash.Hashable(target).String())
		return nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(target).String())
	}
	return tokenInfoRaw, nil
}
