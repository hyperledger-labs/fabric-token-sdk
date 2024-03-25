/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Ledger models a read-only ledger
type Ledger interface {
	GetState(key string) ([]byte, error)
}

// Validator validates a token request
type Validator struct {
	backend driver.Validator
}

// UnmarshalActions returns the actions contained in the serialized token request
func (c *Validator) UnmarshalActions(raw []byte) ([]interface{}, error) {
	return c.backend.UnmarshalActions(raw)
}

// UnmarshallAndVerify unmarshalls the token request and verifies it against the passed ledger and anchor
func (c *Validator) UnmarshallAndVerify(ledger Ledger, binding string, raw []byte) ([]interface{}, map[string][]byte, error) {
	actions, attributes, err := c.backend.VerifyTokenRequestFromRaw(
		ledger.GetState,
		binding,
		raw,
	)
	if err != nil {
		return nil, nil, err
	}

	res := make([]interface{}, len(actions))
	copy(res, actions)
	return res, attributes, nil
}
