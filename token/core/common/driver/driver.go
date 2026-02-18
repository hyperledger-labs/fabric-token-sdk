/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package driver defines common interfaces and type aliases used by token driver implementations.
// These abstractions decouple drivers from concrete service implementations, facilitating
// modularity, testing, and dependency injection.
package driver

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

// MetricsProvider is an alias for the metrics.Provider interface.
type MetricsProvider = metrics.Provider

// TracerProvider is an alias for the trace.TracerProvider interface from OpenTelemetry.
type TracerProvider = trace.TracerProvider

// StorageProvider is an alias for the identity.StorageProvider interface.
type StorageProvider = identity.StorageProvider

// NetworkBinderService is an alias for the identity.NetworkBinderService interface.
type NetworkBinderService = identity.NetworkBinderService

// ConfigService defines the interface for accessing TMS configurations.
type ConfigService interface {
	// ConfigurationFor returns the configuration for the given network, channel, and namespace.
	ConfigurationFor(network, channel, namespace string) (driver.Configuration, error)
}

// IdentityProvider defines the interface for accessing identities.
type IdentityProvider interface {
	// DefaultIdentity returns the default identity.
	DefaultIdentity() driver.Identity
}

// NetworkProvider defines the interface for accessing network instances.
type NetworkProvider interface {
	// GetNetwork returns the network instance for the given network and channel.
	GetNetwork(network, channel string) (*network.Network, error)
}

// VaultProvider defines the interface for accessing vault instances.
type VaultProvider interface {
	// Vault returns the vault instance for the given network, channel, and namespace.
	Vault(network, channel, namespace string) (driver.Vault, error)
}
