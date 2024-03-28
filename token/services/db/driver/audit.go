/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
)

// AuditTransactionDB defines the interface for a database to store the audit records of token transactions.
type AuditTransactionDB = TransactionDB

// AuditDBDriver is the interface for an audit database driver
type AuditDBDriver interface {
	// Open opens an audit database connection
	Open(sp view2.ServiceProvider, tmsID token2.TMSID) (AuditTransactionDB, error)
}
