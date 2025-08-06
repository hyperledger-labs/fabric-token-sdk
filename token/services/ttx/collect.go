/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Actions struct {
	Transfers []*ActionTransfer
}

// ActionTransfer describe a transfer operation
type ActionTransfer struct {
	// From is the sender
	From view.Identity
	// Type of tokens to transfer
	Type token2.Type
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
		if w := ts.WalletManager().OwnerWallet(context.Context(), actionTransfer.From); w != nil {
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
	logger.DebugfContext(context.Context(), "collect local from [%s]", party)

	err := c.tx.Transfer(w, actionTransfer.Type, []uint64{actionTransfer.Amount}, []view.Identity{actionTransfer.Recipient})
	if err != nil {
		return errors.Wrap(err, "failed creating transfer for action")
	}

	// Binds identities
	es := endpoint.GetService(context)
	longTermIdentity, _, _, err := es.Resolve(context.Context(), party)
	if err != nil {
		return errors.Wrapf(err, "cannot resolve long term network identity for [%s]", party)
	}
	if err := c.tx.TokenRequest.BindTo(context.Context(), es, longTermIdentity); err != nil {
		return errors.Wrapf(err, "failed binding to [%s]", party.String())
	}

	return nil
}

func (c *collectActionsView) collectRemote(context view.Context, actionTransfer *ActionTransfer) error {
	party := actionTransfer.From
	logger.DebugfContext(context.Context(), "collect remote from [%s]", party)

	session, err := session2.NewJSON(context, context.Initiator(), party)
	if err != nil {
		return errors.Wrap(err, "failed getting session")
	}

	// Send transaction, actions, action
	txRaw, err := c.tx.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed marshalling transaction")
	}
	if err := session.SendRaw(context.Context(), txRaw); err != nil {
		return errors.Wrap(err, "failed sending transaction")
	}
	if err := session.Send(c.actions); err != nil {
		return errors.Wrapf(err, "failed sending actions")
	}
	if err := session.Send(actionTransfer); err != nil {
		return errors.Wrapf(err, "failed sending action")
	}

	// Wait to receive a content back
	msg, err := session.ReceiveRaw()
	if err != nil {
		return errors.Wrap(err, "failed reading message")
	}
	txPayload := &Payload{
		Transient:    map[string][]byte{},
		TokenRequest: token.NewRequest(nil, ""),
	}
	err = unmarshal(c.tx.NetworkProvider, txPayload, msg)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling reply")
	}

	// Check
	txPayload.TokenRequest.SetTokenService(c.tx.TokenService())
	if err := txPayload.TokenRequest.IsValid(context.Context()); err != nil {
		return errors.Wrap(err, "failed verifying response")
	}

	// Append
	if err = c.tx.appendPayload(txPayload); err != nil {
		return errors.Wrap(err, "failed appending payload")
	}

	// Bind to party
	es := endpoint.GetService(context)
	longTermIdentity, _, _, err := es.Resolve(context.Context(), party)
	if err != nil {
		return errors.Wrapf(err, "cannot resolve long term network identity for [%s]", party)
	}
	if err := txPayload.TokenRequest.BindTo(context.Context(), es, longTermIdentity); err != nil {
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
	txBoxed, err := context.RunView(NewReceiveTransactionView(), view.WithSameContext())
	if err != nil {
		return nil, err
	}
	cctx := txBoxed.(*Transaction)

	// Check that the transaction is valid
	if err := cctx.IsValid(context.Context()); err != nil {
		return nil, errors.WithMessagef(err, "invalid transaction %s", cctx.ID())
	}

	// actions
	s := session2.JSON(context)
	actions := &Actions{}
	if err := s.Receive(actions); err != nil {
		return nil, errors.Wrap(err, "failed receiving actions")
	}

	// action
	action := &ActionTransfer{}
	if err := s.Receive(action); err != nil {
		return nil, errors.Wrap(err, "failed receiving action")
	}

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

	err = context.Session().SendWithContext(context.Context(), response)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending back response")
	}

	return nil, nil
}
