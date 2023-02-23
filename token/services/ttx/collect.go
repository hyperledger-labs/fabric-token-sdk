/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Actions struct {
	Transfers []*ActionTransfer
}

// ActionTransfer describe a transfer operation
type ActionTransfer struct {
	// From is the sender
	From view.Identity
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Recipient is the recipient of the transfer
	Recipient view.Identity
}

type collectActionsView struct {
	tx      *Transaction
	actions *Actions
}

// NewCollectActionsView returns an instance of collectActionsView.
// The view does the following:
// For each action, the view contact the recipient by sending as first message the transaction.
// Then, the view waits for the answer and append it to the transaction.
func NewCollectActionsView(tx *Transaction, actions ...*ActionTransfer) *collectActionsView {
	return &collectActionsView{
		tx: tx,
		actions: &Actions{
			Transfers: actions,
		},
	}
}

func (c *collectActionsView) Call(context view.Context) (interface{}, error) {
	ts := token.GetManagementService(context, token.WithChannel(c.tx.Channel()))

	for _, actionTransfer := range c.actions.Transfers {
		if w := ts.WalletManager().OwnerWalletByIdentity(actionTransfer.From); w != nil {
			if err := c.collectLocal(context, actionTransfer, w); err != nil {
				return nil, err
			}
		} else {
			if err := c.collectRemote(context, actionTransfer); err != nil {
				return nil, err
			}
		}
	}
	return c.tx, nil
}

func (c *collectActionsView) collectLocal(context view.Context, actionTransfer *ActionTransfer, w *token.OwnerWallet) error {
	party := actionTransfer.From
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collect local from [%s]", party)
	}

	err := c.tx.Transfer(w, actionTransfer.Type, []uint64{actionTransfer.Amount}, []view.Identity{actionTransfer.Recipient})
	if err != nil {
		return errors.Wrap(err, "failed creating transfer for action")
	}

	// Binds identities
	if err := c.tx.TokenRequest.BindTo(context, party); err != nil {
		return errors.Wrapf(err, "failed binding to [%s]", party.String())
	}

	return nil
}

func (c *collectActionsView) collectRemote(context view.Context, actionTransfer *ActionTransfer) error {
	party := actionTransfer.From
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collect remote from [%s]", party)
	}

	session, err := context.GetSession(context.Initiator(), party)
	if err != nil {
		return errors.Wrap(err, "failed getting session")
	}

	// Send transaction, actions, action
	txRaw, err := c.tx.Bytes()
	assert.NoError(err)
	assert.NoError(session.Send(txRaw), "failed sending transaction")
	assert.NoError(session.Send(marshalOrPanic(c.actions)), "failed sending actions")
	assert.NoError(session.Send(marshalOrPanic(actionTransfer)), "failed sending transfer action")

	timeout := time.NewTimer(time.Minute)
	defer timeout.Stop()

	// Wait to receive a content back
	ch := session.Receive()
	var msg *view.Message
	select {
	case msg = <-ch:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("collect actions: reply received from [%s]", party)
		}
	case <-timeout.C:
		return errors.Errorf("Timeout from party %s", party)
	}
	if msg.Status == view.ERROR {
		return errors.New(string(msg.Payload))
	}

	txPayload := &Payload{
		Transient:    map[string][]byte{},
		TokenRequest: token.NewRequest(nil, ""),
	}
	err = unmarshal(context, txPayload, msg.Payload)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling reply")
	}

	// Check
	txPayload.TokenRequest.SetTokenService(c.tx.TokenService())
	if err := txPayload.TokenRequest.IsValid(); err != nil {
		return errors.Wrap(err, "failed verifying response")
	}

	// Append
	if err = c.tx.appendPayload(txPayload); err != nil {
		return errors.Wrap(err, "failed appending payload")
	}

	// Bind to party
	if err := txPayload.TokenRequest.BindTo(context, party); err != nil {
		return errors.Wrapf(err, "failed binding to [%s]", party.String())
	}

	return nil
}

type receiveActionsView struct{}

// ReceiveAction runs the receiveActionsView.
// The view does the following: It receives the transaction, the collection of actions, and the requested action.
func ReceiveAction(context view.Context) (*Transaction, *ActionTransfer, error) {
	res, err := context.RunView(&receiveActionsView{})
	if err != nil {
		return nil, nil, err
	}
	result := res.([]interface{})
	return result[0].(*Transaction), result[1].(*ActionTransfer), nil
}

func (r *receiveActionsView) Call(context view.Context) (interface{}, error) {
	// transaction
	txBoxed, err := context.RunView(NewReceiveTransactionView(""), view.WithSameContext())
	if err != nil {
		return nil, err
	}
	cctx := txBoxed.(*Transaction)

	// Check that the transaction is valid
	if err := cctx.IsValid(); err != nil {
		return nil, errors.WithMessagef(err, "invalid transaction %s", cctx.ID())
	}

	// actions
	payload, err := session2.ReadMessageWithTimeout(context.Session(), 120*time.Second)
	if err != nil {
		return nil, err
	}
	actions := &Actions{}
	unmarshalOrPanic(payload, actions)

	// action
	payload, err = session2.ReadMessageWithTimeout(context.Session(), 120*time.Second)
	if err != nil {
		return nil, err
	}
	action := &ActionTransfer{}
	unmarshalOrPanic(payload, action)

	return []interface{}{cctx, action}, nil
}

type collectActionsResponderView struct {
	tx     *Transaction
	action *ActionTransfer
}

// NewCollectActionsResponderView returns an instance of the collectActionsResponderView.
// The view does the following: Sends back the transaction.
func NewCollectActionsResponderView(tx *Transaction, action *ActionTransfer) *collectActionsResponderView {
	return &collectActionsResponderView{tx: tx, action: action}
}

func (s *collectActionsResponderView) Call(context view.Context) (interface{}, error) {
	response, err := s.tx.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling ephemeral transaction")
	}

	err = context.Session().Send(response)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending back response")
	}

	return nil, nil
}

func marshalOrPanic(state interface{}) []byte {
	raw, err := Marshal(state)
	if err != nil {
		panic(fmt.Sprintf("failed marshalling state [%s]", err))
	}
	return raw
}

func unmarshalOrPanic(raw []byte, state interface{}) {
	err := Unmarshal(raw, state)
	if err != nil {
		panic(fmt.Sprintf("failed unmarshalling state [%s]", err))
	}
}
