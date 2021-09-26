/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Payload struct {
	TokenRequest         []byte
	TokenRequestMetadata []byte
	RWSet                []byte
}

func (p *Payload) Bytes() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Payload) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, p)
}

type ActionTransfer struct {
	From      view.Identity
	Type      string
	Amount    uint64
	Recipient view.Identity
}

type Actions struct {
	Transfers []*ActionTransfer
}

func ReceiveAction(context view.Context) (*Transaction, *ActionTransfer, error) {
	// transaction
	txBoxed, err := context.RunView(NewReceiveTransactionView())
	if err != nil {
		return nil, nil, err
	}
	cctx := txBoxed.(*Transaction)

	// actions
	payload, err := session2.ReadMessageWithTimeout(context.Session(), 120*time.Second)
	if err != nil {
		return nil, nil, err
	}
	actions := &Actions{}
	unmarshalOrPanic(payload, actions)

	// action
	payload, err = session2.ReadMessageWithTimeout(context.Session(), 120*time.Second)
	if err != nil {
		return nil, nil, err
	}
	action := &ActionTransfer{}
	unmarshalOrPanic(payload, action)

	return cctx, action, nil
}

type collectActionsView struct {
	tx      *Transaction
	actions *Actions
}

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

	err := c.tx.Transfer(w, actionTransfer.Type, []uint64{actionTransfer.Amount}, []view.Identity{actionTransfer.Recipient})
	if err != nil {
		return errors.Wrap(err, "failed creating transfer for action")
	}

	// Bind identities
	if err := c.tx.TokenRequest.BindTo(context, party); err != nil {
		return errors.Wrapf(err, "failed to bind to [%s]", party.String())
	}

	return nil
}

func (c *collectActionsView) collectRemote(context view.Context, actionTransfer *ActionTransfer) error {
	party := actionTransfer.From

	session, err := context.GetSession(context.Initiator(), party)
	if err != nil {
		return errors.Wrap(err, "failed getting session")
	}

	// Send transaction, actions, action
	// TODO: this can cause problems if the first message sent here is the first message received by the receiver
	txRaw, err := c.tx.Raw()
	assert.NoError(err)
	assert.NoError(session.Send(txRaw), "failed sending transaction")
	assert.NoError(session.Send(marshalOrPanic(c.actions)), "failed sending actions")
	assert.NoError(session.Send(marshalOrPanic(actionTransfer)), "failed sending transfer action")

	// Wait to receive a content back
	ch := session.Receive()
	var msg *view.Message
	select {
	case msg = <-ch:
		logger.Debugf("collect actions: reply received from [%s]", party)
	case <-time.After(60 * time.Second):
		return errors.Errorf("Timeout from party %s", party)
	}
	if msg.Status == view.ERROR {
		return errors.New(string(msg.Payload))
	}

	payload := &Payload{}
	if err := payload.FromBytes(msg.Payload); err != nil {
		return errors.Wrap(err, "failed unmarshalling reply")
	}

	tokenRequest, err := c.tx.TokenService().NewRequestFromBytes(
		c.tx.ID(),
		payload.TokenRequest,
		payload.TokenRequestMetadata,
	)
	if err != nil {
		return errors.Wrap(err, "failed creating token request")
	}

	// Match Request with Metadata
	if err := tokenRequest.Verify(); err != nil {
		return errors.Wrap(err, "failed verifying response")
	}

	// TODO: Match Request with rws

	// append
	if err = c.tx.append(tokenRequest, payload.RWSet); err != nil {
		return errors.Wrap(err, "failed appending payload")
	}

	// Bind identities
	if err := tokenRequest.BindTo(context, party); err != nil {
		return errors.Wrapf(err, "failed binding to [%s]", party.String())
	}

	return nil
}

type collectActionsResponderView struct {
	tx     *Transaction
	action *ActionTransfer
}

func (s *collectActionsResponderView) Call(context view.Context) (interface{}, error) {
	resultsRaw, err := s.tx.Results()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling transaction")
	}

	requestBytes, err := s.tx.TokenRequest.RequestToBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling token request")
	}
	metadataBytes, err := s.tx.TokenRequest.MetadataToBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling token request metadata")
	}
	payload := &Payload{
		TokenRequest:         requestBytes,
		TokenRequestMetadata: metadataBytes,
		RWSet:                resultsRaw,
	}
	reply, err := payload.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling answer")
	}
	err = context.Session().Send(reply)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending back response")
	}

	return nil, nil
}

func NewCollectActionsResponderView(tx *Transaction, action *ActionTransfer) *collectActionsResponderView {
	return &collectActionsResponderView{tx: tx, action: action}
}

func marshalOrPanic(state interface{}) []byte {
	raw, err := json.Marshal(state)
	if err != nil {
		panic(fmt.Sprintf("failed marshalling state [%s]", err))
	}
	return raw
}

func unmarshalOrPanic(raw []byte, state interface{}) {
	err := json.Unmarshal(raw, state)
	if err != nil {
		panic(fmt.Sprintf("failed unmarshalling state [%s]", err))
	}
}
