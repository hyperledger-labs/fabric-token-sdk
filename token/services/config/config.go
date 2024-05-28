/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

type serviceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	GetString(key string) string
	IsSet(key string) bool
	TranslatePath(path string) string
	GetBool(s string) bool
}

type Configuration = driver.Configuration

// configuration is the configuration of a given configuration
type configuration struct {
	cp      configProvider
	keyID   string
	tmsID   driver.TMSID
	version string
}

func NewConfiguration(cp configProvider, version string, keyID string, tmsID driver.TMSID) *configuration {
	return &configuration{
		cp:      cp,
		version: version,
		keyID:   keyID,
		tmsID:   tmsID,
	}
}

func (m *configuration) Version() string {
	return m.version
}

func (m *configuration) ID() driver.TMSID {
	return m.tmsID
}

// TranslatePath translates the passed path relative to the config path
func (m *configuration) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}

// UnmarshalKey takes a single key and unmarshals it into a Struct
func (m *configuration) UnmarshalKey(key string, rawVal interface{}) error {
	return m.cp.UnmarshalKey("token.tms."+m.keyID+"."+key, rawVal)
}

func (m *configuration) GetString(key string) string {
	return m.cp.GetString("token.tms." + m.keyID + "." + key)
}

func (m *configuration) IsSet(key string) bool {
	return m.cp.IsSet("token.tms." + m.keyID + "." + key)
}

// Service model the configuration service for the token sdk
type Service struct {
	cp configProvider

	version  string
	enabled  bool
	tmsCache utils.LazyGetter[map[string]Configuration]
}

// NewService creates a new Service configuration.
func NewService(cp configProvider) *Service {
	version := cp.GetString("token.version")
	if len(version) == 0 {
		version = "v1"
	}
	enabled := cp.GetBool("token.enabled")
	loader := &loader{cp: cp}
	return &Service{
		cp:       cp,
		version:  version,
		enabled:  enabled,
		tmsCache: utils.NewLazyGetter(loader.load),
	}
}

func GetService(sp serviceProvider) (*Service, error) {
	s, err := sp.GetService(reflect.TypeOf((*Service)(nil)))
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting config service")
	}
	return s.(*Service), nil
}

func (m *Service) Version() string {
	return m.version
}

func (m *Service) Enabled() bool {
	return m.enabled
}

// LookupNamespace searches for a configuration configuration that matches the given network and channel, and
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
func (m *Service) ConfigurationFor(network, channel, namespace string) (Configuration, error) {
	tmsConfigs, err := m.configurations()
	if err != nil {
		return nil, err
	}

	for key, config := range tmsConfigs {
		id := config.ID()
		if id.Network == network && id.Channel == channel && id.Namespace == namespace {
			return NewConfiguration(m.cp, m.version, key, id), nil
		}
	}

	return nil, errors.Errorf("no token-sdk configuration for network %s, channel %s, namespace %s", network, channel, namespace)
}

// Configurations returns all configuration configurations.
func (m *Service) Configurations() ([]Configuration, error) {
	tmsConfigs, err := m.configurations()
	if err != nil {
		return nil, err
	}

	var tms []Configuration
	for key, config := range tmsConfigs {
		tms = append(tms, NewConfiguration(m.cp, m.version, key, config.ID()))
	}
	return tms, nil
}

func (m *Service) configurations() (map[string]Configuration, error) {
	return m.tmsCache.Get()
}

type loader struct {
	cp configProvider
}

func (m *loader) load() (map[string]Configuration, error) {
	//load
	var boxedConfig map[interface{}]interface{}
	if err := m.cp.UnmarshalKey("token.tms", &boxedConfig); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configurations")
	}

	tmsConfigs := map[string]Configuration{}
	for k := range boxedConfig {
		id := k.(string)
		tmsID := driver.TMSID{}
		if err := m.cp.UnmarshalKey("token.tms."+id, &tmsID); err != nil {
			return nil, errors.WithMessagef(err, "cannot load token-sdk tms configuration for [%s]", id)
		}
		tmsConfigs[id] = &configuration{cp: m.cp, keyID: id, tmsID: tmsID}
	}
	return tmsConfigs, nil
}
