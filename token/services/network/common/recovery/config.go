/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package recovery

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
)

const (
	// ConfigKeyRecovery is the configuration key for recovery settings
	ConfigKeyRecovery = "network.recovery"
	// ConfigKeyEnabled is the configuration key for enabling recovery
	ConfigKeyEnabled = "network.recovery.enabled"
	// ConfigKeyTTL is the configuration key for transaction TTL
	ConfigKeyTTL = "network.recovery.ttl"
	// ConfigKeyScanInterval is the configuration key for scan interval
	ConfigKeyScanInterval = "network.recovery.scanInterval"
)

// RecoveryConfig holds the recovery configuration from YAML
type RecoveryConfig struct {
	Enabled      bool   `yaml:"enabled"`
	TTL          string `yaml:"ttl"`
	ScanInterval string `yaml:"scanInterval"`
}

// LoadConfig loads the recovery configuration from the TMS configuration
func LoadConfig(cfg *config.Configuration) (Config, error) {
	// Start with defaults
	result := DefaultConfig()

	// Check if recovery configuration exists
	if !cfg.IsSet(ConfigKeyRecovery) {
		return result, nil
	}

	// Unmarshal the recovery configuration
	var recoveryConfig RecoveryConfig
	if err := cfg.UnmarshalKey(ConfigKeyRecovery, &recoveryConfig); err != nil {
		return result, err
	}

	// Apply configuration values
	result.Enabled = recoveryConfig.Enabled

	// Parse TTL if provided
	if recoveryConfig.TTL != "" {
		ttl, err := time.ParseDuration(recoveryConfig.TTL)
		if err != nil {
			return result, err
		}
		result.TTL = ttl
	}

	// Parse scan interval if provided
	if recoveryConfig.ScanInterval != "" {
		interval, err := time.ParseDuration(recoveryConfig.ScanInterval)
		if err != nil {
			return result, err
		}
		result.ScanInterval = interval
	}

	return result, nil
}
