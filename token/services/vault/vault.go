/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type QueryEngine = driver.QueryEngine

type CertificationStorage = driver.CertificationStorage

type TokenVault interface {
	QueryEngine() QueryEngine
	CertificationStorage() CertificationStorage
	DeleteTokens(ns string, ids ...*token.ID) error
}

type Provider interface {
	Vault(network, channel, namespace string) (TokenVault, error)
}

var (
	managerType = reflect.TypeOf((*Provider)(nil))
)

// GetProvider
func GetProvider(sp view.ServiceProvider) (Provider, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token vault provider")
	}
	return s.(Provider), nil
}
