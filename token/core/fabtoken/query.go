/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// HistoryIssuedTokens returns the list of all issued tokens
// An IssuedToken consists of the identity of the token issuer, the token unique identifier
// and information
func (s *Service) HistoryIssuedTokens() (*token2.IssuedTokens, error) {
	return s.QE.ListHistoryIssuedTokens()
}

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
