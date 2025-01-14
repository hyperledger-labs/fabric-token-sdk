/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	errors2 "github.com/pkg/errors"
)

type MultiplexerKeyManagerProvider struct {
	KMPs []KeyManagerProvider
}

func NewMultiplexerKeyManagerProvider(KMPs []KeyManagerProvider) *MultiplexerKeyManagerProvider {
	return &MultiplexerKeyManagerProvider{KMPs: KMPs}
}

func (m *MultiplexerKeyManagerProvider) Get(identityConfig *driver.IdentityConfiguration) (KeyManager, error) {
	if identityConfig == nil {
		return nil, errors2.Errorf("empty identity config passed")
	}
	var errs []error
	for _, p := range m.KMPs {
		km, err := p.Get(identityConfig)
		if err == nil {
			return km, nil
		}
		errs = append(errs, err)
	}
	return nil, errors2.Wrap(errors.Join(errs...), "failed to get a key manager for the passed identity config")
}
