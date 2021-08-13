/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type collectEndorsementsView struct {
	tx *Transaction
}

func NewCollectEndorsementsView(tx *Transaction) view.View {
	return &collectEndorsementsView{tx: tx}
}

func (c *collectEndorsementsView) Call(context view.Context) (interface{}, error) {
	_, err := context.RunView(endorser.NewCollectEndorsementsView(c.tx.tx, c.tx.Endorsers()...))
	if err != nil {
		return nil, errors.WithMessage(err, "failed requesting endorsements")
	}
	if !c.tx.opts.auditor.IsNone() {
		_, err := context.RunView(newAuditingViewInitiator(c.tx))
		if err != nil {
			return nil, errors.WithMessagef(err, "failed requesting auditing from [%s]", c.tx.opts.auditor.String())
		}
		// Cleanup
		session, err := context.GetSession(nil, c.tx.opts.auditor)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting auditor's session")
		}
		session.Close()
	}
	return nil, nil
}

type receiveTransactionView struct{}

func NewReceiveTransactionView() *receiveTransactionView {
	return &receiveTransactionView{}
}

func (f *receiveTransactionView) Call(context view.Context) (interface{}, error) {
	// Wait to receive a transaction back
	ch := context.Session().Receive()

	select {
	case msg := <-ch:
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}
		tx, err := NewTransactionFromBytes(context, msg.Payload)
		if err != nil {
			return nil, err
		}
		return tx, nil
	case <-time.After(240 * time.Second):
		return nil, errors.New("timeout reached")
	}
}
