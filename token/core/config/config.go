/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	TranslatePath(path string) string
}

type Manager struct {
	cp    configProvider
	tms   *driver.TMS
	index int
}

func NewManager(cp configProvider, network, channel, namespace string) (*Manager, error) {
	var tmsConfigs []*driver.TMS
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

func (m *Manager) TMS() *driver.TMS {
	return m.tms
}

func (m *Manager) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}
