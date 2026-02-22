/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricx

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/qe"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
)

var logger = logging.MustGetLogger()

// ledger models the FabricX ledger.
type ledger struct {
	l             *fabric.Ledger
	ch            *fabric.Channel
	keyTranslator translator.KeyTranslator
	executor      qe.QueryStatesExecutor
}

// NewLedger returns a new ledger instance.
func NewLedger(ch *fabric.Channel, keyTranslator translator.KeyTranslator, executor qe.QueryStatesExecutor) *ledger {
	return &ledger{
		ch:            ch,
		l:             ch.Ledger(),
		keyTranslator: keyTranslator,
		executor:      executor,
	}
}

// Status returns the validation code of the transaction with the given ID.
func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	tx, err := l.l.GetTransactionByID(id)
	if err != nil {
		return driver.Unknown, errors.Wrapf(err, "failed to get transaction [%s]", id)
	}
	logger.Debugf("ledger status of [%s] is [%d]", id, tx.ValidationCode())

	switch protoblocktx.Status(tx.ValidationCode()) {
	case protoblocktx.Status_COMMITTED:
		return driver.Valid, nil
	default:
		return driver.Invalid, nil
	}
}

// GetStates returns the values of the given keys in the given namespace.
func (l *ledger) GetStates(ctx context.Context, namespace string, keys ...string) ([][]byte, error) {
	return l.executor.QueryStates(ctx, namespace, keys)
}

// TransferMetadataKey returns the key used to store the metadata of a transfer action.
func (l *ledger) TransferMetadataKey(k string) (string, error) {
	return l.keyTranslator.CreateTransferActionMetadataKey(k)
}
