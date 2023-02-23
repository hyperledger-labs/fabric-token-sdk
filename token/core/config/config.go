/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	IsSet(key string) bool
	TranslatePath(path string) string
}

// TMS is the configuration of a given TMS
type TMS struct {
	cp  configProvider
	id  string
	tms *config.TMS
}

func NewTMS(cp configProvider, id string, tms *config.TMS) *TMS {
	return &TMS{cp: cp, id: id, tms: tms}
}

// TMS returns the full TMS
func (m *TMS) TMS() *config.TMS {
	return m.tms
}

// TranslatePath translates the passed path relative to the config path
func (m *TMS) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}

// UnmarshalKey takes a single key and unmarshals it into a Struct
func (m *TMS) UnmarshalKey(key string, rawVal interface{}) error {
	return m.cp.UnmarshalKey("token.tms."+m.id+"."+key, rawVal)
}

func (m *TMS) IsSet(key string) bool {
	return m.cp.IsSet("token.tms." + m.id + "." + key)
}

// TokenSDK is the configuration of the TokenSDK
type TokenSDK struct {
	cp configProvider

	tmsConfigsLock sync.RWMutex
	tmsConfigs     map[string]*config.TMS
}

// NewTokenSDK creates a new TokenSDK configuration.
func NewTokenSDK(cp configProvider) *TokenSDK {
	return &TokenSDK{cp: cp}
}

// LookupNamespace searches for a TMS configuration that matches the given network and channel, and
// return its namespace.
// If no matching configuration is found, an error is returned.
// If multiple matching configurations are found, an error is returned.
func (m *TokenSDK) LookupNamespace(network, channel string) (string, error) {
	tmsConfigs, err := m.tmss()
	if err != nil {
		return "", err
	}

	var hits []*config.TMS
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

// GetTMS returns a TMS configuration for the given network, channel, and namespace.
func (m *TokenSDK) GetTMS(network, channel, namespace string) (*TMS, error) {
	tmsConfigs, err := m.tmss()
	if err != nil {
		return nil, err
	}

	for id, config := range tmsConfigs {
		if config.Network == network && config.Channel == channel && config.Namespace == namespace {
			return NewTMS(m.cp, id, config), nil
		}
	}

	return nil, errors.Errorf("no token-sdk configuration for network %s, channel %s, namespace %s", network, channel, namespace)
}

// GetTMSs returns all TMS configurations.
func (m *TokenSDK) GetTMSs() ([]*TMS, error) {
	tmsConfigs, err := m.tmss()
	if err != nil {
		return nil, err
	}

	var tms []*TMS
	for id, config := range tmsConfigs {
		tms = append(tms, NewTMS(m.cp, id, config))
	}
	return tms, nil
}

func (m *TokenSDK) tmss() (map[string]*config.TMS, error) {
	// check if available
	m.tmsConfigsLock.RLock()
	if m.tmsConfigs != nil {
		m.tmsConfigsLock.RUnlock()
		return m.tmsConfigs, nil
	}
	m.tmsConfigsLock.RUnlock()

	m.tmsConfigsLock.Lock()
	defer m.tmsConfigsLock.Unlock()

	// check again
	if m.tmsConfigs != nil {
		return m.tmsConfigs, nil
	}

	//load
	var boxedConfig map[interface{}]interface{}
	if err := m.cp.UnmarshalKey("token.tms", &boxedConfig); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configurations")
	}

	tmsConfigs := map[string]*config.TMS{}
	for k := range boxedConfig {
		id := k.(string)
		var tmsConfig *config.TMS
		if err := m.cp.UnmarshalKey("token.tms."+id, &tmsConfig); err != nil {
			return nil, errors.WithMessagef(err, "cannot load token-sdk tms configuration for [%s]", id)
		}
		tmsConfigs[id] = tmsConfig
	}
	m.tmsConfigs = tmsConfigs
	return m.tmsConfigs, nil
}
