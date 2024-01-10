/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func UnmarshallIssuedToken(raw []byte) (*token.IssuedToken, error) {
	t := &token.IssuedToken{}
	err := json.Unmarshal(raw, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}
