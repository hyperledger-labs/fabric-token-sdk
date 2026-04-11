/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

// Service configurations
func (m *Service) ResetConfigurations() error {
	return m.configurationsHolder.Reset()
}

func (m *Service) ConfigurationsInternal() (map[string]*Configuration, error) {
	return m.configurations()
}

func (m *Service) AddConfigurationInternal(cp Provider, raw []byte) error {
	return m.addConfiguration(cp, raw)
}

// Configuration configurations
func (m *Configuration) SetValidators(validators []Validator) {
	m.validators = validators
}

func NewConfigurationInternal(cp Provider, keyID string, tmsID driver.TMSID) *Configuration {
	return NewConfiguration(cp, keyID, tmsID)
}
