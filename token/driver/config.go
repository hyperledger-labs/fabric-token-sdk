/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// Configuration provides functions to access the configuration of a given TMS
type Configuration interface {
	// ID identities the TMS this configuration refers to.
	ID() TMSID
	// IsSet checks to see if the key has been set in any of the data locations
	IsSet(key string) bool
	// UnmarshalKey takes a single key and unmarshals it into a struct.
	// The key must be relative to the TMS this configuration refers to.
	UnmarshalKey(key string, rawVal interface{}) error
	// GetString returns the value associated with the key as a string
	GetString(key string) string
	// TranslatePath translates the passed path relative to the config path
	TranslatePath(path string) string
}
