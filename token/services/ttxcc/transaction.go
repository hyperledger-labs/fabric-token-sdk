/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxcc

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type Payload struct {
	Id        network.TxID
	Network   string
	Channel   string
	Namespace string
	Signer    view.Identity
	Transient network.TransientMap

	TokenRequest *token.Request

	Envelope *network.Envelope
}

type Transaction struct {
	*Payload
	SP   view2.ServiceProvider
	Opts *txOptions
}

// NewAnonymousTransaction returns a new anonymous token transaction customized with the passed opts
func NewAnonymousTransaction(sp view.Context, opts ...TxOption) (*Transaction, error) {
	txOpts, err := compile(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}
	return NewTransaction(
		sp,
		network.GetInstance(sp, txOpts.network, txOpts.channel).AnonymousIdentity(),
		opts...,
	)
}

func NewTransaction(sp view.Context, signer view.Identity, opts ...TxOption) (*Transaction, error) {
	txOpts, err := compile(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}

	tms := token.GetManagementService(
		sp,
		token.WithNetwork(txOpts.network),
		token.WithChannel(txOpts.channel),
		token.WithNamespace(txOpts.namespace),
	)

	id := &network.TxID{Creator: signer}
	tr, err := tms.NewRequest(network.GetInstance(sp, tms.Network(), tms.Channel()).ComputeTxID(id))
	if err != nil {
		return nil, errors.WithMessage(err, "failed init token request")
	}

	tx := &Transaction{
		Payload: &Payload{
			Signer:       signer,
			TokenRequest: tr,
			Envelope:     nil,
			Id:           *id,
			Network:      tms.Network(),
			Channel:      tms.Channel(),
			Namespace:    tms.Namespace(),
			Transient:    map[string][]byte{},
		},
		SP:   sp,
		Opts: txOpts,
	}
	sp.OnError(tx.Release)
	return tx, nil
}

func NewTransactionFromBytes(sp view.Context, nw string, channel string, raw []byte) (*Transaction, error) {
	// TODO: remove the need of network by introducing custom Pyaload unmarshalling
	tx := &Transaction{
		Payload: &Payload{
			Envelope:  network.GetInstance(sp, nw, channel).NewEnvelope(),
			Transient: map[string][]byte{},
		},
		SP: sp,
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err := json.Unmarshal(raw, tx.Payload); err != nil {
		return nil, err
	}

	logger.Debugf("unmarshalling tx, id [%s]", tx.Payload.Id.String())

	tx.TokenRequest.SetTokenService(
		token.GetManagementService(sp,
			token.WithNetwork(tx.Network()),
			token.WithChannel(tx.Channel()),
			token.WithNamespace(tx.Namespace()),
		),
	)
	if tx.ID() != tx.TokenRequest.ID() {
		return nil, errors.Errorf("invalid transaction, transaction ids do not match [%s][%s]", tx.ID(), tx.TokenRequest.ID())
	}

	if tx.Envelope != nil {
		if err := tx.setEnvelope(tx.Envelope); err != nil {
			return nil, err
		}
	}
	sp.OnError(tx.Release)
	return tx, nil
}

func ReceiveTransaction(context view.Context) (*Transaction, error) {
	logger.Debugf("receive a new transaction...")

	txBoxed, err := context.RunView(NewReceiveTransactionView(""))
	if err != nil {
		return nil, err
	}

	cctx, ok := txBoxed.(*Transaction)
	if !ok {
		return nil, errors.Errorf("received transaction of wrong type [%T]", cctx)
	}
	logger.Debugf("received transaction with id [%s]", cctx.ID())

	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "received", "tx", cctx.ID())

	return cctx, nil
}

// ID returns the ID of this transaction. It is equal to the underlying Fabric transaction's ID.
func (t *Transaction) ID() string {
	return network.GetInstance(t.SP, t.Network(), t.Channel()).ComputeTxID(&t.Payload.Id)
}

func (t *Transaction) Network() string {
	return t.Payload.Network
}

func (t *Transaction) Channel() string {
	return t.Payload.Channel
}

func (t *Transaction) Namespace() string {
	return t.Payload.Namespace
}

func (t *Transaction) Bytes() ([]byte, error) {
	logger.Debugf("marshalling tx, id [%s]", t.Payload.Id.String())
	return json.Marshal(t.Payload)
}

// Issue appends a new Issue operation to the TokenRequest inside this transaction
func (t *Transaction) Issue(wallet *token.IssuerWallet, receiver view.Identity, typ string, q uint64, opts ...token.IssueOption) error {
	_, err := t.TokenRequest.Issue(wallet, receiver, typ, q, opts...)
	return err
}

// Transfer appends a new Transfer operation to the TokenRequest inside this transaction
func (t *Transaction) Transfer(wallet *token.OwnerWallet, typ string, values []uint64, owners []view.Identity, opts ...token.TransferOption) error {
	_, err := t.TokenRequest.Transfer(wallet, typ, values, owners, opts...)
	return err
}

func (t *Transaction) Redeem(wallet *token.OwnerWallet, typ string, value uint64, opts ...token.TransferOption) error {
	return t.TokenRequest.Redeem(wallet, typ, value, opts...)
}

func (t *Transaction) Outputs() (*token.OutputStream, error) {
	return t.TokenRequest.Outputs()
}

func (t *Transaction) Inputs() (*token.InputStream, error) {
	return t.TokenRequest.Inputs()
}

// Verify checks that the transaction is well formed.
// This means checking that the embedded TokenRequest is valid.
func (t *Transaction) Verify() error {
	return t.TokenRequest.Verify()
}

func (t *Transaction) IsValid() error {
	return t.TokenRequest.IsValid()
}

func (t *Transaction) MarshallToAudit() ([]byte, error) {
	return t.TokenRequest.MarshallToAudit()
}

// Selector returns the default token selector for this transaction
func (t *Transaction) Selector() (token.Selector, error) {
	return t.TokenService().SelectorManager().NewSelector(t.ID())
}

func (t *Transaction) Release() {
	logger.Debugf("releasing resources for tx [%s]", t.ID())
	if err := t.TokenService().SelectorManager().Unlock(t.ID()); err != nil {
		logger.Warnf("failed releasing tokens locked by [%s], [%s]", t.ID(), err)
	}
}

func (t *Transaction) TokenService() *token.ManagementService {
	return token.GetManagementService(
		t.SP,
		token.WithNetwork(t.Network()),
		token.WithChannel(t.Channel()),
		token.WithNamespace(t.Namespace()),
	)
}

func (t *Transaction) ApplicationMetadata(k string) []byte {
	return t.TokenRequest.ApplicationMetadata(k)
}

func (t *Transaction) SetApplicationMetadata(k string, v []byte) {
	t.TokenRequest.SetApplicationMetadata(k, v)
}

func (t *Transaction) storeTransient() error {
	logger.Debugf("Storing transient for [%s]", t.ID())
	raw, err := t.TokenRequest.MetadataToBytes()
	if err != nil {
		return err
	}

	if err := t.Payload.Transient.Set("zkat", raw); err != nil {
		return err
	}

	return network.GetInstance(t.SP, t.Network(), t.Channel()).StoreTransient(t.ID(), t.Payload.Transient)
}

func (t *Transaction) setEnvelope(envelope *network.Envelope) error {
	t.Payload.Id.Nonce = envelope.Nonce()
	t.Payload.Id.Creator = envelope.Creator()
	t.Envelope = envelope

	return nil
}

func (t *Transaction) appendPayload(payload *Payload) error {
	// TODO: change this
	t.Payload.TokenRequest = payload.TokenRequest
	t.Payload.Transient = payload.Transient
	return nil

	// for _, bytes := range payload.Request.Issues {
	//	t.Payload.Request.Issues = append(t.Payload.Request.Issues, bytes)
	// }
	// for _, bytes := range payload.Request.Transfers {
	//	t.Payload.Request.Transfers = append(t.Payload.Request.Transfers, bytes)
	// }
	// for _, info := range payload.TokenInfos {
	//	t.Payload.TokenInfos = append(t.Payload.TokenInfos, info)
	// }
	// for _, issueMetadata := range payload.Metadata.Issues {
	//	t.Payload.Metadata.Issues = append(t.Payload.Metadata.Issues, issueMetadata)
	// }
	// for _, transferMetadata := range payload.Metadata.Transfers {
	//	t.Payload.Metadata.Transfers = append(t.Payload.Metadata.Transfers, transferMetadata)
	// }
	//
	// for key, value := range payload.Transient {
	//	for _, v := range value {
	//		if err := t.Set(key, v); err != nil {
	//			return err
	//		}
	//	}
	// }
	// return nil
}
