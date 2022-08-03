/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Ledger models a read-only ledger
type Ledger interface {
	GetState(key string) ([]byte, error)
}

// Validator validates a token request
type Validator struct {
	backend tokenapi.Validator
}

// UnmarshallAndVerify unmarshalls the token request and verifies it against the passed ledger and anchor
func (c *Validator) UnmarshallAndVerify(ledger Ledger, binding string, raw []byte) ([]interface{}, error) {
	actions, err := c.backend.VerifyTokenRequestFromRaw(func(key string) ([]byte, error) {
		return ledger.GetState(key)
	}, binding, raw)
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, len(actions))
	copy(res, actions)
	return res, nil
}
