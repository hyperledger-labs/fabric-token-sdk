/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
)

const (
	// Type is the configuration key for the manager type.
	Type = config.Type
)

// ManagerType is the type of the manager.
type ManagerType = config.ManagerType

// Configuration is the interface that wraps the GetString method.
//
//go:generate counterfeiter -o mock/configuration.go -fake-name Configuration . Configuration
type Configuration interface {
	// GetString returns the string value for the given key.
	GetString(k string) string
}

const (
	// Notification is the notification manager type.
	Notification ManagerType = "notification"
)

// NewListenerManagerConfig returns a new listener manager configuration.
func NewListenerManagerConfig(configuration Configuration) *serviceListenerManagerConfig {
	return &serviceListenerManagerConfig{c: configuration}
}

// serviceListenerManagerConfig is the listener manager configuration.
type serviceListenerManagerConfig struct {
	c Configuration
}

// Type returns the manager type.
func (c *serviceListenerManagerConfig) Type() ManagerType {
	if v := ManagerType(c.c.GetString(Type)); len(v) > 0 {
		return v
	}

	return Notification
}
