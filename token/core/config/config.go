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
	ID      string `yaml:"id"`
	MSPType string `yaml:"mspType"`
	MSPID   string `yaml:"mspID"`
	Path    string `yaml:"path"`
}

type Wallets struct {
	Certifiers []*Identity `yaml:"certifiers,omitempty"`
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
