/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

const (
	// EndorsementAckPrefix is the prefix for the endorsement ACKs.
	EndorsementAckPrefix = "ttx.endorse.ack"
)

// TransactionInfo contains the transaction info.
type TransactionInfo struct {
	// EndorsementAcks contains the endorsement ACKs received at time of dissemination.
	EndorsementAcks map[string][]byte
}

type txOwner struct {
	sp    view2.ServiceProvider
	tms   *token.ManagementService
	owner *owner.Owner
}

// NewOwner returns a new owner service.
func NewOwner(sp view2.ServiceProvider, tms *token.ManagementService) *txOwner {
	return &txOwner{
		sp:    sp,
		tms:   tms,
		owner: owner.New(sp, tms),
	}
}

// NewQueryExecutor returns a new query executor.
// The query executor is used to execute queries against the token transaction DB.
// The function `Done` on the query executor must be called when it is no longer needed.
func (a *txOwner) NewQueryExecutor() *owner.QueryExecutor {
	return a.owner.NewQueryExecutor()
}

// Append adds a new transaction to the token transaction database.
func (a *txOwner) Append(tx *Transaction) error {
	return a.owner.Append(tx)
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *txOwner) TransactionInfo(txID string) (*TransactionInfo, error) {
	// Load transaction endorsement ACKs
	k := kvs.GetService(a.sp)
	acks := make(map[string][]byte)
	it, err := k.GetByPartialCompositeID(EndorsementAckPrefix, []string{txID})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed loading ack for [%s]", txID)
	}
	defer it.Close()
	for {
		if !it.HasNext() {
			break
		}
		var ack []byte
		key, err := it.Next(&ack)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed loading ack for [%s]", txID)
		}

		objectType, attrs, err := kvs.SplitCompositeKey(key)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed splitting composite key for [%s]", txID)
		}
		if objectType != EndorsementAckPrefix {
			return nil, errors.Errorf("unexpected object type [%s]", objectType)
		}
		if len(attrs) != 2 {
			return nil, errors.Errorf("unexpected number of attributes [%d]", len(attrs))
		}
		acks[attrs[1]] = ack
	}
	if len(acks) == 0 {
		return nil, errors.Errorf("no ack found for [%s]", txID)
	}

	return &TransactionInfo{
		EndorsementAcks: acks,
	}, nil
}

func (a *txOwner) appendTransactionEndorseAck(tx *Transaction, id view.Identity, sigma []byte) error {
	k := kvs.GetService(a.sp)
	ackKey, err := kvs.CreateCompositeKey(EndorsementAckPrefix, []string{tx.ID(), id.UniqueID()})
	if err != nil {
		return errors.Wrap(err, "failed creating composite key")
	}
	if err := k.Put(ackKey, sigma); err != nil {
		return errors.WithMessagef(err, "failed storing ack for [%s:%s]", tx.ID(), id.UniqueID())
	}
	return nil
}
