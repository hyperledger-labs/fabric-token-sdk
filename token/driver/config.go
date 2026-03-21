/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// Configuration provides methods for accessing the configuration of a specific Token Management Service (TMS).
// It abstracts the configuration's source and structure, allowing the TMS to retrieve settings
// such as identifiers, paths, and values in different formats.
//
//go:generate counterfeiter -o mock/configuration.go -fake-name Configuration . Configuration
type Configuration interface {
	// ID returns the unique identifier of the TMS to which this configuration applies.
	ID() TMSID

	// IsSet checks if a specific configuration key is defined.
	IsSet(key string) bool

	// UnmarshalKey decodes the configuration value associated with a key into a provided struct or interface.
	// The key is typically relative to the TMS configuration block.
	UnmarshalKey(key string, rawVal interface{}) error

	// GetString retrieves the value for a key as a string.
	GetString(key string) string

	// GetBool retrieves the value for a key as a boolean.
	GetBool(key string) bool

	// TranslatePath converts a relative configuration path into an absolute file system path.
	TranslatePath(path string) string
}
