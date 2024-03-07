/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

type Config interface {
	TranslatePath(path string) string
}

type IdentityConfig struct {
	Config Config
	TMS    *config2.TMS
}

func NewIdentityConfig(Config Config, TMS *config2.TMS) *IdentityConfig {
	return &IdentityConfig{Config: Config, TMS: TMS}
}

func (i *IdentityConfig) CacheSizeForOwnerID(id string) (int, error) {
	for _, owner := range i.TMS.TMS().Wallets.Owners {
		if owner.ID == id {
			return owner.CacheSize, nil
		}
	}
	return -1, nil
}

func (i *IdentityConfig) TranslatePath(path string) string {
	return i.Config.TranslatePath(path)
}

func (i *IdentityConfig) IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error) {
	tmsConfig := i.TMS.TMS()
	if tmsConfig.Wallets == nil {
		tmsConfig.Wallets = &config.Wallets{}
	}

	switch role {
	case driver.IssuerRole:
		return tmsConfig.Wallets.Issuers, nil
	case driver.AuditorRole:
		return tmsConfig.Wallets.Auditors, nil
	case driver.OwnerRole:
		return tmsConfig.Wallets.Owners, nil
	case driver.CertifierRole:
		return tmsConfig.Wallets.Certifiers, nil
	default:
		return nil, errors.Errorf("unknown role [%d]", role)
	}
}
