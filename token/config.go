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

// NewConfiguration returns a new instance of Configuration
func NewConfiguration(cm driver.Configuration) *Configuration {
	return &Configuration{cm: cm}
}

// IsSet checks to see if the key has been set in any of the data locations
func (m *Configuration) IsSet(key string) bool {
	return m.cm.IsSet(key)
}

// UnmarshalKey takes a single key and unmarshals it into a Struct
func (m *Configuration) UnmarshalKey(key string, rawVal interface{}) error {
	return m.cm.UnmarshalKey(key, rawVal)
}

// GetValidationConfig returns the validation configuration
func (m *Configuration) GetValidationConfig() (driver.ValidationConfig, error) {
	config := driver.ValidationConfig{
		MaxTokenPayloadSize:  2 * 1024 * 1024,
		MaxTokenOutputsPerTx: 1000,
		MaxBulkDeleteSize:    10000,
		MaxWalletIDSize:      1024,
		MaxOwnerRawSize:      16 * 1024,
		MaxIssuerRawSize:     16 * 1024,
		MaxTokenRequestSize:  2 * 1024 * 1024,
		MaxActionCount:       1000,
	}
	if m.cm.IsSet("validation") {
		if err := m.cm.UnmarshalKey("validation", &config); err != nil {
			return config, err
		}
	}

	return config, nil
}
