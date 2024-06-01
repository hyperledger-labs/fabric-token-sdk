/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Configuration manages the configuration of the token-sdk
type Configuration struct {
	cm driver.Configuration
}

// IsSet checks to see if the key has been set in any of the data locations
func (m *Configuration) IsSet(key string) bool {
	return m.cm.IsSet(key)
}

// UnmarshalKey takes a single key and unmarshals it into a Struct
func (m *Configuration) UnmarshalKey(key string, rawVal interface{}) error {
	return m.cm.UnmarshalKey(key, rawVal)
}
