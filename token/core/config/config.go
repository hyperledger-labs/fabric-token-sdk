/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	TranslatePath(path string) string
}

// TMSProvider is a config provider for a specific TMS
type TMSProvider struct {
	cp    configProvider
	tms   *driver.TMS
	index int
}

// NewTMSProvider creates a new TMSProvider for a specific TMS.
// If no matching configuration is found, an error is returned.
func NewTMSProvider(cp configProvider, network, channel, namespace string) (*TMSProvider, error) {
	var tmsConfigs []*driver.TMS
	if err := cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configuration")
	}

	for i, config := range tmsConfigs {
		if config.Network == network && config.Channel == channel && config.Namespace == namespace {
			return &TMSProvider{
				tms:   config,
				index: i,
				cp:    cp,
			}, nil
		}
	}

	return nil, errors.Errorf("no token-sdk configuration for network %s, channel %s, namespace %s", network, channel, namespace)
}

// TMS returns the TMS configuration this provider is associated with.
func (m *TMSProvider) TMS() *driver.TMS {
	return m.tms
}

// TranslatePath translates the passed path relative to the config path
func (m *TMSProvider) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}

// Manager manages the configuration of the token-sdk.
type Manager struct {
	cp configProvider
}

// NewManager creates a new Manager.
func NewManager(cp configProvider) (*Manager, error) {
	var tmsConfigs []*driver.TMS
	if err := cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configuration")
	}
	return &Manager{cp: cp}, nil
}

// SearchTMS searches for a TMS configuration that matches the given network and channel, and
// return its namespace.
// If no matching configuration is found, an error is returned.
// If multiple matching configurations are found, an error is returned.
func (m *Manager) SearchTMS(network, channel string) (string, error) {
	var tmsConfigs []*driver.TMS
	if err := m.cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return "", errors.WithMessagef(err, "cannot load token-sdk configuration")
	}

	var hits []*driver.TMS
	for _, config := range tmsConfigs {
		if config.Network == network && config.Channel == channel {
			hits = append(hits, config)
		}
	}
	if len(hits) == 1 {
		return hits[0].Namespace, nil
	}
	if len(hits) == 0 {
		return "", errors.Errorf("no token-sdk configuration for network %s, channel %s", network, channel)
	}
	return "", errors.Errorf("multiple token-sdk configurations for network %s, channel %s", network, channel)
}
