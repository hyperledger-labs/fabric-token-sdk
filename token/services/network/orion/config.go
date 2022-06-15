/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	TranslatePath(path string) string
}

type Manager struct {
	cp    configProvider
	tms   *config.TMS
	index int
}

func NewManager(cp configProvider, network, channel, namespace string) (*Manager, error) {
	var tmsConfigs []*config.TMS
	if err := cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configuration")
	}

	for i, config := range tmsConfigs {
		if config.Network == network && config.Channel == channel && config.Namespace == namespace {
			return &Manager{
				tms:   config,
				index: i,
				cp:    cp,
			}, nil
		}
	}

	return nil, errors.Errorf("no token-sdk configuration for network %s, channel %s, namespace %s", network, channel, namespace)
}

func (m *Manager) TMS() *config.TMS {
	return m.tms
}

func (m *Manager) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}

func IsCustodian(cp configProvider) (bool, error) {
	var tmsConfigs []*TMS
	if err := cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return false, errors.WithMessagef(err, "cannot load token-sdk configuration")
	}

	for _, config := range tmsConfigs {
		if config.Orion == nil {
			continue
		}
		logger.Debugf("config: %v", config.Orion.Custodian)
		if config.Orion.Custodian.Enabled {
			return true, nil
		}
	}
	return false, nil
}

func GetCustodian(cp configProvider, network string) (string, error) {
	var tmsConfigs []*TMS
	if err := cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return "", errors.WithMessagef(err, "cannot load token-sdk configuration")
	}
	for _, config := range tmsConfigs {
		if config.Network == network {
			if config.Orion == nil {
				return "", errors.Errorf("no orion configuration for network %s", network)
			}
			return config.Orion.Custodian.ID, nil
		}
	}

	return "", errors.Errorf("no token-sdk configuration for network %s", network)
}
