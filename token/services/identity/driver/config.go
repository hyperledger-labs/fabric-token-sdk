/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// ConfiguredIdentity describes an identity entry parsed from configuration.
//
// Fields:
//   - ID: the unique identifier for the identity
//   - Default: whether this identity should be considered the default
//   - Path: file-system path or location where credentials/configuration live
//   - CacheSize: per-identity cache size used by the identity service; 0 means no cache
//   - Opts: provider-specific options (opaque)
type ConfiguredIdentity struct {
	ID        string      `yaml:"id"`
	Default   bool        `yaml:"default,omitempty"`
	Path      string      `yaml:"path"`
	CacheSize int         `yaml:"cacheSize"`
	Opts      interface{} `yaml:"opts,omitempty"`
}

// String returns the identity's ID and satisfies fmt.Stringer.
func (i *ConfiguredIdentity) String() string {
	return i.ID
}

// Config is a read-only view over identity service configuration.
// Implementors provide lookup helpers used by the identity service to resolve configured
// identities and to normalize configured paths.
type Config interface {
	// CacheSizeForOwnerID returns the cache size to be used for the given owner wallet.
	// If not defined for the given id, the function returns -1.
	CacheSizeForOwnerID(id string) int

	// TranslatePath maps a configured relative or templated path to an absolute
	// runtime path. Implementations should resolve variables and perform any
	// environment-specific transformations required to locate credential files.
	TranslatePath(path string) string
}
