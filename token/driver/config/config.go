/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

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
	Certifiers []*Identity `yaml:"certifiers,omitempty"`
	Owners     []*Identity `yaml:"owners,omitempty"`
	Issuers    []*Identity `yaml:"issuers,omitempty"`
	Auditors   []*Identity `yaml:"auditors,omitempty"`
}

type TMS struct {
	Network       string         `yaml:"network,omitempty"`
	Channel       string         `yaml:"channel,omitempty"`
	Namespace     string         `yaml:"namespace,omitempty"`
	Certification *Certification `yaml:"certification,omitempty"`
	Wallets       *Wallets       `yaml:"wallets,omitempty"`
}

type Token struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	TMS     []*TMS `yaml:"tms,omitempty"`
}

type Manager interface {
	TMS() *TMS
	TranslatePath(path string) string
}
