/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *service) ListTokens() (*token2.UnspentTokens, error) {
	return s.qe.ListUnspentTokens()
}

func (s *service) HistoryIssuedTokens() (*token2.IssuedTokens, error) {
	return s.qe.ListHistoryIssuedTokens()
}

func (s *service) DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, view.Identity, error) {
	tok := &token2.Token{}
	if err := json.Unmarshal(outputRaw, tok); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token")
	}

	tokInfo := &TokenInformation{}
	if err := tokInfo.Deserialize(tokenInfoRaw); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token information")
	}

	return tok, tokInfo.Issuer, nil
}
