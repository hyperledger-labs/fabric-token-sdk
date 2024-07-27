/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	db "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TxStatus is the status of a transaction
type TxStatus = db.TxStatus

type QueryEngine interface {
	driver.QueryEngine
	GetStatus(txID string) (TxStatus, string, error)
}

type CertificationStorage = driver.CertificationStorage

type Vault interface {
	QueryEngine() QueryEngine
	CertificationStorage() CertificationStorage
	DeleteTokens(toDelete ...*token.ID) error
}

type Provider interface {
	Vault(network, channel, namespace string) (Vault, error)
}
