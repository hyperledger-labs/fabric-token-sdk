/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/driver"
	msp2 "github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
)

const (
	MSPType = "idemix"
)

type IdentityLoader struct{}

func (i *IdentityLoader) Load(manager driver.Manager, c config.MSP) error {
	conf, err := msp2.GetLocalMspConfigWithType(manager.Config().TranslatePath(c.Path), nil, c.MSPID, c.MSPType)
	if err != nil {
		return errors.Wrapf(err, "failed reading idemix msp configuration from [%s]", manager.Config().TranslatePath(c.Path))
	}
	provider, err := NewProviderWithAnyPolicy(conf, manager.ServiceProvider())
	if err != nil {
		return errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", manager.Config().TranslatePath(c.Path))
	}
	manager.AddDeserializer(provider)
	cacheSize := manager.CacheSize()
	if c.CacheSize > 0 {
		cacheSize = c.CacheSize
	}
	manager.AddMSP(c.ID, c.MSPType, provider.EnrollmentID(), NewIdentityCache(provider.Identity, cacheSize, nil).Identity)
	logger.Debugf("added %s msp for id %s with cache of size %d", c.MSPType, c.ID+"@"+provider.EnrollmentID(), cacheSize)

	return nil
}

type FolderIdentityLoader struct {
	*IdentityLoader
}

func (f *FolderIdentityLoader) Load(manager driver.Manager, c config.MSP) error {
	entries, err := os.ReadDir(manager.Config().TranslatePath(c.Path))
	if err != nil {
		logger.Warnf("failed reading from [%s]: [%s]", manager.Config().TranslatePath(c.Path), err)
		return errors.Wrapf(err, "failed reading from [%s]", manager.Config().TranslatePath(c.Path))
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()

		if err := f.IdentityLoader.Load(manager, config.MSP{
			ID:      id,
			MSPType: MSPType,
			MSPID:   id,
			Path:    filepath.Join(manager.Config().TranslatePath(c.Path), id),
		}); err != nil {
			return errors.WithMessagef(err, "failed to load Idemix MSP configuration [%s]", id)
		}
	}
	return nil
}
