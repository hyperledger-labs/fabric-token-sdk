/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type InteractiveCertification struct {
	IDs []string `yaml:"ids,omitempty"`
}

type Certification struct {
	Interactive *InteractiveCertification `yaml:"interactive,omitempty"`
}

type Identity struct {
	ID      string `yaml:"id"`
	Default bool   `yaml:"default,omitempty"`
	Type    string `yaml:"type"`
	Path    string `yaml:"path"`
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

type ConfigManager interface {
	TMS() *TMS
	TranslatePath(path string) string
}
