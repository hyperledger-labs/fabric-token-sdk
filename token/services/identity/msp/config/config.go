/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	config2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

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
