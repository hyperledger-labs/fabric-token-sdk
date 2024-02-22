/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	config2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config interface {
	// CacheSizeForOwnerID returns the cache size to be used for the given owner wallet.
	// If not defined, the function returns -1
	CacheSizeForOwnerID(id string) (int, error)
	TranslatePath(path string) string
	IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error)
}

// ToBCCSPOpts converts the passed opts to `config.BCCSP`
func ToBCCSPOpts(opts interface{}) (*config2.BCCSP, error) {
	if opts == nil {
		return nil, nil
	}
	out, err := yaml.Marshal(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "faild to marshal [%v]", opts)
	}
	mspOpts := &config2.MSPOpts{}
	if err := yaml.Unmarshal(out, mspOpts); err != nil {
		return nil, errors.Wrapf(err, "faild to unmarshal [%v] to BCCSP options", opts)
	}
	return mspOpts.BCCSP, nil
}
