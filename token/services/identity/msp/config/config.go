/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type MSPOpts struct {
	BCCSP *BCCSP `yaml:"BCCSP,omitempty"`
}

type BCCSP struct {
	Default string            `yaml:"Default,omitempty"`
	SW      *SoftwareProvider `yaml:"SW,omitempty"`
	PKCS11  *PKCS11           `yaml:"PKCS11,omitempty"`
}

type SoftwareProvider struct {
	Hash     string `yaml:"Hash,omitempty"`
	Security int    `yaml:"Security,omitempty"`
}

type PKCS11 struct {
	// Default algorithms when not specified (Deprecated?)
	Security int    `yaml:"Security"`
	Hash     string `yaml:"Hash"`

	// PKCS11 options
	Library        string         `yaml:"Library"`
	Label          string         `yaml:"Label"`
	Pin            string         `yaml:"Pin"`
	SoftwareVerify bool           `yaml:"SoftwareVerify,omitempty"`
	Immutable      bool           `yaml:"Immutable,omitempty"`
	AltID          string         `yaml:"AltId,omitempty"`
	KeyIDs         []KeyIDMapping `yaml:"KeyIds,omitempty" mapstructure:"KeyIds"`
}

type KeyIDMapping struct {
	SKI string `yaml:"SKI,omitempty"`
	ID  string `yaml:"ID,omitempty"`
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

type Config interface {
	// CacheSizeForOwnerID returns the cache size to be used for the given owner wallet.
	// If not defined, the function returns -1
	CacheSizeForOwnerID(id string) (int, error)
	TranslatePath(path string) string
	IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error)
}

// ToBCCSPOpts converts the passed opts to `config.BCCSP`
func ToBCCSPOpts(opts interface{}) (*BCCSP, error) {
	if opts == nil {
		return nil, nil
	}
	out, err := yaml.Marshal(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "faild to marshal [%v]", opts)
	}
	mspOpts := &MSPOpts{}
	if err := yaml.Unmarshal(out, mspOpts); err != nil {
		return nil, errors.Wrapf(err, "faild to unmarshal [%v] to BCCSP options", opts)
	}
	return mspOpts.BCCSP, nil
}
