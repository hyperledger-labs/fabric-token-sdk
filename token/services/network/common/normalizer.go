/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"

// Configuration models the configuration provider
type Configuration interface {
	// LookupNamespace searches for a TMS configuration that matches the given network and channel, and
	// return its namespace.
	// If no matching configuration is found, an error is returned.
	// If multiple matching configurations are found, an error is returned.
	LookupNamespace(network, channel string) (string, error)

	ConfigurationFor(network, channel, namespace string) (config.Configuration, error)
}
