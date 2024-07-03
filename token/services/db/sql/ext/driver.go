/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ext

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Extension interface {
	GetSchema() string
}

type TokenDBExtension interface {
	Extension
	Delete(tx *sql.Tx, txID string, index uint64, deletedBy string) error
	StoreToken(tx *sql.Tx, tr driver.TokenRecord, owners []string) error
	DeleteTokens(tx *sql.Tx, deletedBy string, ids ...*token.ID) error
}
