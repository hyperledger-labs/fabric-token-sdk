/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	db "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type QueryEngine interface {
	driver.QueryEngine
	GetStatus(txID string) (TxStatus, string, error)
}

type CertificationStorage = driver.CertificationStorage

// TxStatus is the status of a transaction
type TxStatus = db.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown = db.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = db.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = db.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = db.Deleted
)

type Vault interface {
	QueryEngine() QueryEngine
	CertificationStorage() CertificationStorage
	DeleteTokens(toDelete ...*token.ID) error
}

type Provider interface {
	Vault(network, channel, namespace string) (Vault, error)
}

var (
	managerType = reflect.TypeOf((*Provider)(nil))
)

// GetProvider returns the registered instance of Provider from the passed service provider
func GetProvider(sp view.ServiceProvider) (Provider, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token vault provider")
	}
	return s.(Provider), nil
}
