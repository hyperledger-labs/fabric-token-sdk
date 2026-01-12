/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type ConfiguredIdentity struct {
	ID        string      `yaml:"id"`
	Default   bool        `yaml:"default,omitempty"`
	Path      string      `yaml:"path"`
	CacheSize int         `yaml:"cacheSize"`
	Type      string      `yaml:"type,omitempty"`
	Opts      interface{} `yaml:"opts,omitempty"`
}

func (i *ConfiguredIdentity) String() string {
	return i.ID
}

type Config interface {
	// CacheSizeForOwnerID returns the cache size to be used for the given owner wallet.
	// If not defined, the function returns -1
	CacheSizeForOwnerID(id string) int
	TranslatePath(path string) string
	IdentitiesForRole(role IdentityRoleType) ([]ConfiguredIdentity, error)
}
