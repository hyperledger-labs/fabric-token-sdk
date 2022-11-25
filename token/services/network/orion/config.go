/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/pkg/errors"
)

type configProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	TranslatePath(path string) string
}

func IsCustodian(cp configProvider) (bool, error) {
	tmsConfigs, err := tmss(cp)
	if err != nil {
		return false, err
	}
	for _, config := range tmsConfigs {
		if config.Orion == nil {
			continue
		}
		logger.Debugf("config: %v", config.Orion.Custodian)
		if config.Orion.Custodian.Enabled {
			return true, nil
		}
	}
	return false, nil
}

func GetCustodian(cp configProvider, network string) (string, error) {
	tmsConfigs, err := tmss(cp)
	if err != nil {
		return "", err
	}
	for _, config := range tmsConfigs {
		if config.Network == network {
			if config.Orion == nil {
				return "", errors.Errorf("no orion configuration for network %s", network)
			}
			return config.Orion.Custodian.ID, nil
		}
	}

	return "", errors.Errorf("no token-sdk configuration for network %s", network)
}

func tmss(cp configProvider) (map[string]*TMS, error) {
	var boxedConfig map[interface{}]interface{}
	if err := cp.UnmarshalKey("token.tms", &boxedConfig); err != nil {
		return nil, errors.WithMessagef(err, "cannot load token-sdk configurations")
	}

	tmsConfigs := map[string]*TMS{}
	for k := range boxedConfig {
		id := k.(string)
		var tmsConfig *TMS
		if err := cp.UnmarshalKey("token.tms."+id, &tmsConfig); err != nil {
			return nil, errors.WithMessagef(err, "cannot load token-sdk tms configuration for [%s]", id)
		}
		tmsConfigs[id] = tmsConfig
	}
	return tmsConfigs, nil
}
