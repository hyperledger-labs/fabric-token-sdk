/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Locker interface {
	Lock(id *token2.ID, txID string, reclaim bool) (string, error)
	UnlockIDs(id ...*token2.ID) []*token2.ID
	UnlockByTxID(txID string)
	IsLocked(id *token2.ID) bool
}

type ExtendedSelector struct {
	Selector token.Selector
	Lock     Locker
}

func (s *ExtendedSelector) Select(ownerFilter token.OwnerFilter, q, tokenType string) ([]*token2.ID, token2.Quantity, error) {
	return s.Selector.Select(ownerFilter, q, tokenType)
}

func (s *ExtendedSelector) Unselect(id ...*token2.ID) {
	if s.Lock != nil {
		s.Lock.UnlockIDs(id...)
	}
}
