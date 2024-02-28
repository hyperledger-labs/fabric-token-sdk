/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

// TxStatus is the status of a transaction
type TxStatus string

const (
	// Unknown is the status of a transaction that is unknown
	Unknown TxStatus = "Unknown"
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending TxStatus = "Pending"
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed TxStatus = "Confirmed"
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted TxStatus = "Deleted"
)
