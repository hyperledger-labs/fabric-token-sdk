/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	RootKey = "token"
	TMSKey  = "tms"
)

var (
	TMSPath     = config.Join(RootKey, TMSKey)
	VersionPath = config.Join(RootKey, "version")
	EnabledPath = config.Join(RootKey, "enabled")
)

type Provider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	GetString(key string) string
	IsSet(key string) bool
	TranslatePath(path string) string
	GetBool(s string) bool
	MergeConfig(raw []byte) error
}

// Service model the configuration service for the token sdk
type Service struct {
	cp Provider

	version  string
	enabled  bool
	tmsCache lazy.Getter[map[string]*Configuration]
}

// NewService creates a new Service configuration.
func NewService(cp Provider) *Service {
	version := cp.GetString(VersionPath)
	if len(version) == 0 {
		version = "v1"
	}
	enabled := cp.GetBool(EnabledPath)
	loader := &loader{cp: cp}
	return &Service{
		cp:       cp,
		version:  version,
		enabled:  enabled,
		tmsCache: lazy.NewGetter(loader.load),
	}
}

func (m *Service) Version() string {
	return m.version
}

func (m *Service) Enabled() bool {
	return m.enabled
}

// LookupNamespace searches for a configuration that matches the given network and channel, and
// return its namespace.
// If no matching configuration is found, an error is returned.
// If multiple matching configurations are found, an error is returned.
func (m *Service) LookupNamespace(network, channel string) (string, error) {
	tmsConfigs, err := m.configurations()
	if err != nil {
		return "", err
	}

	var hits []driver.TMSID
	for _, config := range tmsConfigs {
		id := config.ID()
		if id.Network == network && id.Channel == channel {
			hits = append(hits, id)
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

// ConfigurationFor returns a configuration for the given network, channel, and namespace.
func (m *Service) ConfigurationFor(network, channel, namespace string) (*Configuration, error) {
	tmsConfigs, err := m.configurations()
	if err != nil {
		return nil, err
	}

	for key, config := range tmsConfigs {
		id := config.ID()
		if id.Network == network && id.Channel == channel && id.Namespace == namespace {
			return NewConfiguration(m.cp, key, id), nil
		}
	}

	return nil, errors.Errorf("no token-sdk configuration for network %s, channel %s, namespace %s", network, channel, namespace)
}

// Configurations returns all configuration configurations.
func (m *Service) Configurations() ([]*Configuration, error) {
	tmsConfigs, err := m.configurations()
	if err != nil {
		return nil, errors.Wrapf(err, "failed loading configurations")
	}
	return collections.Values(tmsConfigs), nil
}

func (m *Service) AddConfiguration(raw []byte) error {
	// Do the following:
	// - parse raw as a yaml stream
	// - extract the configuration
	// - validate it making sure it contains a new TMS
	// If all good, accept the new TMS
	// Updates to an existing TMS should be rejected.
	err := m.cp.MergeConfig(raw)
	if err != nil {
		return err
	}
	return nil
}

func (m *Service) configurations() (map[string]*Configuration, error) {
	return m.tmsCache.Get()
}

type loader struct {
	cp Provider
}

func (m *loader) load() (map[string]*Configuration, error) {
	// load
	var boxedConfig map[interface{}]interface{}
	if err := m.cp.UnmarshalKey(TMSPath, &boxedConfig); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configurations")
	}

	tmsConfigs := map[string]*Configuration{}
	for k := range boxedConfig {
		id := k.(string)
		tmsID := driver.TMSID{}
		if err := m.cp.UnmarshalKey(config.Join(TMSPath, id), &tmsID); err != nil {
			return nil, errors.WithMessagef(err, "cannot load token-sdk tms configuration for [%s]", id)
		}
		tmsConfigs[id] = NewConfiguration(m.cp, id, tmsID)
		if err := tmsConfigs[id].Validate(); err != nil {
			return nil, errors.WithMessagef(err, "cannot load token-sdk configuration for [%s]", id)
		}
	}
	return tmsConfigs, nil
}
