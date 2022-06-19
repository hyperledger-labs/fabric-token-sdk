/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// StoreEnvelope stores the transaction envelope locally
func StoreEnvelope(context view.Context, tx *Transaction) error {
	agent := metrics.Get(context)

	agent.EmitKey(0, "ttx", "start", "acceptViewParseRWS", tx.ID())
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("parse rws for id [%s]", tx.ID())
	}
	env := tx.Payload.Envelope
	backend := network.GetInstance(context, tx.Network(), tx.Channel())
	rws, err := backend.GetRWSet(tx.ID(), env.Results())
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset for tx [%s]", tx.ID())
	}
	rws.Done()
	agent.EmitKey(0, "ttx", "end", "acceptViewParseRWS", tx.ID())

	agent.EmitKey(0, "ttx", "start", "acceptViewStoreEnv", tx.ID())
	rawEnv, err := env.Bytes()
	if err != nil {
		return errors.WithMessagef(err, "failed marshalling tx env [%s]", tx.ID())
	}

	if err := backend.StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return errors.WithMessagef(err, "failed storing tx env [%s]", tx.ID())
	}
	agent.EmitKey(0, "ttx", "end", "acceptViewStoreEnv", tx.ID())

	agent.EmitKey(0, "ttx", "size", "acceptViewEnvelopeSize", tx.ID(), strconv.Itoa(len(rawEnv)))

	return nil
}

// StoreTransactionRecords stores the transaction records extracted from the passed transaction to the
// token transaction db
func StoreTransactionRecords(context view.Context, tx *Transaction) error {
	return NewOwner(context, tx.TokenRequest.TokenService).Append(tx)
}
