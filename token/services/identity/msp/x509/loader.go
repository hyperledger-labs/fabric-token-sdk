/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/driver"
	"github.com/pkg/errors"
)

const (
	MSPType       = "bccsp"
	BCCSPOptField = "BCCSP"
)

type IdentityLoader struct{}

func (i *IdentityLoader) Load(manager driver.Manager, c config.MSP) error {
	var bccspOpts *config.BCCSP
	if c.Opts != nil {
		logger.Debugf("Options [%v]", c.Opts)
		bccspOptsBoxed, ok := c.Opts[BCCSPOptField]
		if ok {
			var err error
			bccspOpts, err = ToBCCSPOpts(bccspOptsBoxed)
			if err != nil {
				return errors.Wrapf(err, "failed to unmarshal BCCSP opts")
			}
			logger.Debugf("Options unmarshalled [%v]", bccspOpts)
		}
	}

	// Try without "msp"
	rootPath := filepath.Join(manager.Config().TranslatePath(c.Path))
	provider, err := NewProviderWithBCCSPConfig(rootPath, "", c.MSPID, manager.SignerService(), bccspOpts)
	if err != nil {
		logger.Warnf("failed reading bccsp msp configuration from [%s]: [%s]", rootPath, err)
		// Try with "msp"
		provider, err = NewProviderWithBCCSPConfig(filepath.Join(rootPath, "msp"), "", c.MSPID, manager.SignerService(), bccspOpts)
		if err != nil {
			logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
				rootPath, filepath.Join(rootPath, "msp"), err,
			)
			return errors.WithMessagef(err, "failed to load BCCSP MSP configuration [%s]", c.ID)
		}
	}

	manager.AddDeserializer(provider)
	manager.AddMSP(c.ID, c.MSPType, provider.EnrollmentID(), provider.Identity)

	// set default
	defaultIdentity, _, err := provider.Identity(nil)
	if err != nil {
		return errors.WithMessagef(err, "failed to get default identity for [%s]", c.MSPID)
	}
	defaultSigningIdentity, err := provider.SerializedIdentity()
	if err != nil {
		return errors.WithMessagef(err, "failed to get default signing identity for [%s]", c.MSPID)
	}
	manager.SetDefaultIdentity(c.ID, defaultIdentity, defaultSigningIdentity)

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
			Opts:    c.Opts,
		}); err != nil {
			return errors.WithMessagef(err, "failed to load BCCSP MSP configuration [%s]", id)
		}
	}
	return nil
}

func ToBCCSPOpts(boxed interface{}) (*config.BCCSP, error) {
	raw, err := yaml.Marshal(boxed)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal")
	}
	opts := &config.BCCSP{}
	if err := yaml.Unmarshal(raw, opts); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal")
	}
	return opts, nil
}
