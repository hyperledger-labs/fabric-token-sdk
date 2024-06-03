/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type Config interface {
	TranslatePath(path string) string
	UnmarshalKey(key string, rawVal interface{}) error
}

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

type Wallets struct {
	DefaultCacheSize int         `yaml:"defaultCacheSize,omitempty"`
	Certifiers       []*Identity `yaml:"certifiers,omitempty"`
	Owners           []*Identity `yaml:"owners,omitempty"`
	Issuers          []*Identity `yaml:"issuers,omitempty"`
	Auditors         []*Identity `yaml:"auditors,omitempty"`
}

type IdentityConfig struct {
	Config  Config
	Wallets *Wallets
}

func NewIdentityConfig(config Config) (*IdentityConfig, error) {
	wallets := &Wallets{}
	if err := config.UnmarshalKey("wallets", wallets); err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling wallets")
	}
	return &IdentityConfig{Config: config, Wallets: wallets}, nil
}

func (i *IdentityConfig) CacheSizeForOwnerID(id string) int {
	for _, owner := range i.Wallets.Owners {
		if owner.ID == id {
			return owner.CacheSize
		}
	}
	return -1
}

func (i *IdentityConfig) DefaultCacheSize() int {
	return i.Wallets.DefaultCacheSize
}

func (i *IdentityConfig) TranslatePath(path string) string {
	return i.Config.TranslatePath(path)
}

func (i *IdentityConfig) IdentitiesForRole(role driver.IdentityRole) ([]*Identity, error) {
	switch role {
	case driver.IssuerRole:
		return i.Wallets.Issuers, nil
	case driver.AuditorRole:
		return i.Wallets.Auditors, nil
	case driver.OwnerRole:
		return i.Wallets.Owners, nil
	case driver.CertifierRole:
		return i.Wallets.Certifiers, nil
	default:
		return nil, errors.Errorf("unknown role [%d]", role)
	}
}
