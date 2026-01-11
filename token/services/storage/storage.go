/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storage

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/common"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

type TxStatus = dbdriver.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown = dbdriver.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = dbdriver.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = dbdriver.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = dbdriver.Deleted
)

var TxStatusMessage = dbdriver.TxStatusMessage

type (
	TransactionRecord     = dbdriver.TransactionRecord
	TokenRequestRecord    = dbdriver.TokenRequestRecord
	WalletID              = dbdriver.WalletID
	IdentityConfiguration = dbdriver.IdentityConfiguration
	WalletStore           = dbdriver.WalletStore
	IdentityStore         = dbdriver.IdentityStore
	KeyStore              = dbdriver.KeyStore
	StatusEvent           = common.StatusEvent
)
