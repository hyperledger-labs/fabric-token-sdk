/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// StoreEnvelope stores the transaction envelope locally
func StoreEnvelope(context view.Context, tx *Transaction) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("parse rws for id [%s]", tx.ID())
	}
	backend := network.GetInstance(context, tx.Network(), tx.Channel())
	if err := backend.StoreEnvelope(tx.Payload.Envelope); err != nil {
		return errors.WithMessagef(err, "failed storing tx env [%s]", tx.ID())
	}

	return nil
}

// StoreTransactionRecords stores the transaction records extracted from the passed transaction to the
// token transaction db
func StoreTransactionRecords(context view.Context, tx *Transaction) error {
	return NewOwner(context, tx.TokenRequest.TokenService).Append(tx)
}

// RunView runs passed view within the passed context and using the passed options in a separate goroutine
func RunView(context view.Context, view view.View, opts ...view.RunViewOption) {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf("panic in RunView: %v", r)
		}
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Debugf("panic in RunView: %v", r)
			}
		}()
		_, err := context.RunView(view, opts...)
		if err != nil {
			logger.Errorf("failed to run view: %s", err)
		}
	}()
}
