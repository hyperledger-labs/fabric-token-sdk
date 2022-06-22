/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	TranslatePath(path string) string
}

// TMS is the configuration of a given TMS
type TMS struct {
	cp  configProvider
	tms *config.TMS
}

// TMS returns the full TMS
func (m *TMS) TMS() *config.TMS {
	return m.tms
}

// TranslatePath translates the passed path relative to the config path
func (m *TMS) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}

// TokenSDK is the configuration of the TokenSDK
type TokenSDK struct {
	cp configProvider
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
	var tmsConfigs []*config.TMS
	if err := m.cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return "", errors.WithMessagef(err, "cannot load token-sdk configuration")
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
	var tmsConfigs []*config.TMS
	if err := m.cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configuration")
	}

	for _, config := range tmsConfigs {
		if config.Network == network && config.Channel == channel && config.Namespace == namespace {
			return &TMS{
				tms: config,
				cp:  m.cp,
			}, nil
		}
	}

	return nil, errors.Errorf("no token-sdk configuration for network %s, channel %s, namespace %s", network, channel, namespace)
}

// GetTMSs returns all TMS configurations.
func (m *TokenSDK) GetTMSs() ([]*TMS, error) {
	var tmsConfigs []*config.TMS
	if err := m.cp.UnmarshalKey("token.tms", &tmsConfigs); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configuration")
	}

	var tms []*TMS
	for _, config := range tmsConfigs {
		tms = append(tms, &TMS{
			tms: config,
			cp:  m.cp,
		})
	}
	return tms, nil
}
