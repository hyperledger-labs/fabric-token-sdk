/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Configuration is the configuration of a given configuration
type Configuration struct {
	cp    Provider
	keyID string
	tmsID driver.TMSID
}

func NewConfiguration(cp Provider, keyID string, tmsID driver.TMSID) *Configuration {
	return &Configuration{
		cp:    cp,
		keyID: keyID,
		tmsID: tmsID,
	}
}

func (m *Configuration) Validate() error {
	// check TMS ID
	if len(m.tmsID.Network) == 0 {
		return errors.New("missing network id")
	}
	if len(m.tmsID.Namespace) == 0 {
		return errors.New("missing namespace id")
	}

	return nil
}

func (m *Configuration) ID() driver.TMSID {
	return m.tmsID
}

// TranslatePath translates the passed path relative to the config path
func (m *Configuration) TranslatePath(path string) string {
	return m.cp.TranslatePath(path)
}

// UnmarshalKey takes a single key and unmarshals it into a Struct
func (m *Configuration) UnmarshalKey(key string, rawVal interface{}) error {
	return m.cp.UnmarshalKey(config.Join(TMSPath, m.keyID, key), rawVal)
}

func (m *Configuration) GetString(key string) string {
	return m.cp.GetString(config.Join(TMSPath, m.keyID, key))
}

func (m *Configuration) GetBool(key string) bool {
	return m.cp.GetBool(config.Join(TMSPath, m.keyID, key))
}

func (m *Configuration) IsSet(key string) bool {
	return m.cp.IsSet(config.Join(TMSPath, m.keyID, key))
}
