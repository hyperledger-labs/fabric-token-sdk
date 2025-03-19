/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
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

// NewCollectActionsView returns an instance of CollectActionsView.
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
	return context.RunView(&CollectActionsView{
		tx:              c.tx,
		actions:         c.actions,
		tmsProvider:     token.GetManagementServiceProvider(context),
		endpointService: view2.NewEndpointService(driver.GetEndpointService(context)),
	})
}

type CollectActionsView struct {
	tx      *Transaction
	actions *Actions

	tmsProvider     *token.ManagementServiceProvider
	endpointService *view2.EndpointService
}

func (c *CollectActionsView) Call(context view.Context) (interface{}, error) {
	ts, err := c.tmsProvider.GetManagementService(token.WithChannel(c.tx.Channel()))
	if err != nil {
		return nil, errors.Wrapf(err, "tms not found")
	}

	for _, actionTransfer := range c.actions.Transfers {
		if w := ts.WalletManager().OwnerWallet(actionTransfer.From); w != nil {
			if err := c.collectLocal(actionTransfer, w); err != nil {
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

func (c *CollectActionsView) collectLocal(actionTransfer *ActionTransfer, w *token.OwnerWallet) error {
	party := actionTransfer.From
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collect local from [%s]", party)
	}

	err := c.tx.Transfer(w, actionTransfer.Type, []uint64{actionTransfer.Amount}, []view.Identity{actionTransfer.Recipient})
	if err != nil {
		return errors.Wrap(err, "failed creating transfer for action")
	}

	// Binds identities
	longTermIdentity, _, _, err := c.endpointService.Resolve(party)
	if err != nil {
		return errors.Wrapf(err, "cannot resolve long term network identity for [%s]", party)
	}
	if err := c.tx.TokenRequest.BindTo(c.endpointService, longTermIdentity); err != nil {
		return errors.Wrapf(err, "failed binding to [%s]", party.String())
	}

	return nil
}

func (c *CollectActionsView) collectRemote(context view.Context, actionTransfer *ActionTransfer) error {
	party := actionTransfer.From
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collect remote from [%s]", party)
	}

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
	if err := txPayload.TokenRequest.IsValid(); err != nil {
		return errors.Wrap(err, "failed verifying response")
	}

	// Append
	if err = c.tx.appendPayload(txPayload); err != nil {
		return errors.Wrap(err, "failed appending payload")
	}

	// Bind to party
	longTermIdentity, _, _, err := c.endpointService.Resolve(party)
	if err != nil {
		return errors.Wrapf(err, "cannot resolve long term network identity for [%s]", party)
	}
	if err := txPayload.TokenRequest.BindTo(c.endpointService, longTermIdentity); err != nil {
		return errors.Wrapf(err, "failed binding to [%s]", party.String())
	}

	return nil
}

type CollectActionsViewFactory struct {
	tmsProvider     *token.ManagementServiceProvider
	endpointService *view2.EndpointService
}

func NewCollectActionsViewFactory(
	tmsProvider *token.ManagementServiceProvider,
	endpointService *view2.EndpointService,
) *CollectActionsViewFactory {
	return &CollectActionsViewFactory{
		tmsProvider:     tmsProvider,
		endpointService: endpointService,
	}
}

func (f *CollectActionsViewFactory) New(tx *Transaction, actions ...*ActionTransfer) (*CollectActionsView, error) {
	return &CollectActionsView{
		tx: tx,
		actions: &Actions{
			Transfers: actions,
		},
		tmsProvider:     f.tmsProvider,
		endpointService: f.endpointService,
	}, nil
}

type ReceiveActionsView struct {
	kvss            *kvs.KVS
	tmsProvider     *token.ManagementServiceProvider
	networkProvider *network.Provider
}

// ReceiveAction runs the ReceiveActionsView.
// The view does the following: It receives the transaction, the collection of actions, and the requested action.
func ReceiveAction(context view.Context) (*Transaction, *ActionTransfer, error) {
	res, err := context.RunView(&ReceiveActionsView{
		kvss:            utils.MustGet(context.GetService(&kvs.KVS{})).(*kvs.KVS),
		tmsProvider:     token.GetManagementServiceProvider(context),
		networkProvider: network.GetProvider(context),
	})
	if err != nil {
		return nil, nil, err
	}
	result := res.([]interface{})
	return result[0].(*Transaction), result[1].(*ActionTransfer), nil
}

func (r *ReceiveActionsView) Call(context view.Context) (interface{}, error) {
	// transaction
	txBoxed, err := context.RunView(NewReceiveTransactionView(r.kvss, r.tmsProvider, r.networkProvider), view.WithSameContext())
	if err != nil {
		return nil, err
	}
	cctx := txBoxed.(*Transaction)

	// Check that the transaction is valid
	if err := cctx.IsValid(); err != nil {
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

	err = context.Session().Send(response)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending back response")
	}

	return nil, nil
}
