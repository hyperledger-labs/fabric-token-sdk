/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
)

type Identity struct {
	ID        string      `yaml:"id"`
	Default   bool        `yaml:"default,omitempty"`
	Path      string      `yaml:"path"`
	CacheSize int         `yaml:"cacheSize"`
	Type      string      `yaml:"type,omitempty"`
	Opts      interface{} `yaml:"opts,omitempty"`
}

func (i *Identity) String() string {
	return i.ID
}

type Config interface {
	// CacheSizeForOwnerID returns the cache size to be used for the given owner wallet.
	// If not defined, the function returns -1
	CacheSizeForOwnerID(id string) (int, error)
	TranslatePath(path string) string
	IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error)
}
