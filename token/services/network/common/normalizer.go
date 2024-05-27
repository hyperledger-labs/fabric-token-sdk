/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

// NamespaceFinder models a namespace finder
type NamespaceFinder interface {
	// LookupNamespace searches for a TMS configuration that matches the given network and channel, and
	// return its namespace.
	// If no matching configuration is found, an error is returned.
	// If multiple matching configurations are found, an error is returned.
	LookupNamespace(network, channel string) (string, error)
}
