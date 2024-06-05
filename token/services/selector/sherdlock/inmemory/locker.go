/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package inmemory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Locker interface {
	Lock(id *token.ID, txID string, reclaim bool) (string, error)
	UnlockByTxID(txID string)
}

type Vault interface {
	Status(id string) (int, error)
}

type locker struct {
	Locker
}

func NewLocker(l Locker) *locker {
	return &locker{Locker: l}
}

func (l *locker) Lock(tokenID *token.ID, consumerTxID transaction.ID) error {
	_, err := l.Locker.Lock(tokenID, consumerTxID, false)
	return err
}

func (l *locker) UnlockByTxID(txID transaction.ID) error {
	l.Locker.UnlockByTxID(txID)
	return nil
}
