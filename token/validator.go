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
func (c *Validator) UnmarshallAndVerify(ledger Ledger, anchor string, raw []byte) ([]interface{}, error) {
	actions, _, err := c.backend.VerifyTokenRequestFromRaw(func(key string) ([]byte, error) {
		return ledger.GetState(key)
	}, anchor, raw)
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, len(actions))
	copy(res, actions)
	return res, nil
}

// UnmarshallAndVerifyWithMetadata behaves as UnmarshallAndVerify. In addition, it returns the metadata extracts from the token request
// in the form of map.
func (c *Validator) UnmarshallAndVerifyWithMetadata(ledger Ledger, anchor string, raw []byte) ([]interface{}, map[string][]byte, error) {
	actions, meta, err := c.backend.VerifyTokenRequestFromRaw(func(key string) ([]byte, error) {
		return ledger.GetState(key)
	}, anchor, raw)
	if err != nil {
		return nil, nil, err
	}

	res := make([]interface{}, len(actions))
	copy(res, actions)
	return res, meta, nil
}
