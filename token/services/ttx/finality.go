/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

const (
	finalityTimeout = 10 * time.Minute
	pollingTimeout  = 500 * time.Millisecond
)

type finalityDB interface {
	AddStatusListener(txID string, ch chan common.StatusEvent)
	DeleteStatusListener(txID string, ch chan common.StatusEvent)
	GetStatus(txID string) (TxStatus, string, error)
	GetStatuses(txIDs ...driver.TxID) (driver3.StatusResponseIterator, error)
}

// NewFinalityView returns an instance of the batchedFinalityView.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func NewFinalityView(tx *Transaction, opts ...TxOption) *batchedFinalityView {
	return NewFinalityWithOpts(append([]TxOption{WithTransactions(tx)}, opts...)...)
}

func NewFinalityWithOpts(opts ...TxOption) *batchedFinalityView {
	return newBatchedFinalityWithOpts(opts...)
}
