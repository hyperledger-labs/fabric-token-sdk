/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"

// TxStatus is the status of a transaction
type TxStatus = driver.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown TxStatus = driver.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending TxStatus = driver.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed TxStatus = driver.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted TxStatus = driver.Deleted
)
