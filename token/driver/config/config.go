/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

const (
	DefaultCacheSize = 3
)

type InteractiveCertification struct {
	IDs []string `yaml:"ids,omitempty"`
}

type Certification struct {
	Interactive *InteractiveCertification `yaml:"interactive,omitempty"`
}

type Identity struct {
	ID        string      `yaml:"id"`
	Default   bool        `yaml:"default,omitempty"`
	Path      string      `yaml:"path"`
	CacheSize int         `yaml:"cacheSize"`
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

type TMS struct {
	Network       string         `yaml:"network,omitempty"`
	Channel       string         `yaml:"channel,omitempty"`
	Namespace     string         `yaml:"namespace,omitempty"`
	Driver        string         `yaml:"driver,omitempty"`
	Certification *Certification `yaml:"certification,omitempty"`
	Wallets       *Wallets       `yaml:"wallets,omitempty"`
}

func (t *TMS) GetOwnerWallet(id string) *Identity {
	if t.Wallets == nil {
		return nil
	}
	owners := t.Wallets.Owners
	if len(owners) == 0 {
		return nil
	}
	for _, owner := range owners {
		if owner.ID == id {
			return owner
		}
	}
	return nil
}

func (t *TMS) GetWalletDefaultCacheSize() int {
	if t.Wallets == nil {
		return DefaultCacheSize
	}
	return t.Wallets.DefaultCacheSize
}

type Manager interface {
	TMS() *TMS
	TranslatePath(path string) string
	IsSet(key string) bool
	UnmarshalKey(key string, rawVal interface{}) error
}
