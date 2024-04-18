/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
)

// ConfigManager manages the configuration of the token-sdk
type ConfigManager struct {
	cm config.Manager
}

// Certifiers returns the list of certifier ids.
func (m *ConfigManager) Certifiers() []string {
	return m.cm.TMS().Certification.Interactive.IDs
}

// UnmarshalKey takes a single key and unmarshals it into a Struct
func (m *ConfigManager) UnmarshalKey(key string, rawVal interface{}) error {
	return m.cm.UnmarshalKey(key, rawVal)
}
