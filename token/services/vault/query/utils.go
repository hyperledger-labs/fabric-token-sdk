/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package query

import (
	"encoding/json"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func UnmarshallFabtoken(raw []byte) (*token2.Token, error) {
	t := &token2.Token{}
	err := json.Unmarshal(raw, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func UnmarshallIssuedToken(raw []byte) (*token2.IssuedToken, error) {
	t := &token2.IssuedToken{}
	err := json.Unmarshal(raw, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}
