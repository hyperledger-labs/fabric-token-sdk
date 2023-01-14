/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/asn1"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type Payload struct {
	TxID      network.TxID
	ID        string
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
	Opts *TxOptions
}

// NewAnonymousTransaction returns a new anonymous token transaction customized with the passed opts
func NewAnonymousTransaction(sp view.Context, opts ...TxOption) (*Transaction, error) {
	txOpts, err := compile(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}
	return NewTransaction(
		sp,
		network.GetInstance(sp, txOpts.Network, txOpts.Channel).AnonymousIdentity(),
		opts...,
	)
}

// NewTransaction returns a new token transaction customized with the passed opts that will be signed by the passed signer.
// A valid signer is a signer that the target network recognizes as so. For example, in case of fabric, the signer must be a valid fabric identity.
// If the passed signer is nil, then the default identity is used.
func NewTransaction(sp view.Context, signer view.Identity, opts ...TxOption) (*Transaction, error) {
	txOpts, err := compile(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}

	tms := token.GetManagementService(
		sp,
		token.WithNetwork(txOpts.Network),
		token.WithChannel(txOpts.Channel),
		token.WithNamespace(txOpts.Namespace),
	)

	nw := network.GetInstance(sp, tms.Network(), tms.Channel())
	if signer.IsNone() {
		signer = nw.LocalMembership().DefaultIdentity()
	}
	txID := &network.TxID{Creator: signer}
	id := nw.ComputeTxID(txID)
	tr, err := tms.NewRequest(id)
	if err != nil {
		return nil, errors.WithMessage(err, "failed init token request")
	}

	tx := &Transaction{
		Payload: &Payload{
			Signer:       signer,
			TokenRequest: tr,
			Envelope:     nil,
			TxID:         *txID,
			ID:           id,
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

func NewTransactionFromBytes(sp view.Context, raw []byte) (*Transaction, error) {
	tx := &Transaction{
		Payload: &Payload{
			Transient:    map[string][]byte{},
			TokenRequest: token.NewRequest(nil, ""),
		},
		SP: sp,
	}

	if err := unmarshal(sp, tx.Payload, raw); err != nil {
		return nil, err
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("unmarshalling tx, id [%s]", tx.Payload.TxID.String())
	}

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

	// if tx.Envelope != nil {
	// 	if err := tx.setEnvelope(tx.Envelope); err != nil {
	// 		return nil, err
	// 	}
	// }
	sp.OnError(tx.Release)
	return tx, nil
}

func ReceiveTransaction(context view.Context, opts ...TxOption) (*Transaction, error) {
	opt, err := compile(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to parse options")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("receive a new transaction...")
	}

	txBoxed, err := context.RunView(NewReceiveTransactionView(""), view.WithSameContext())
	if err != nil {
		return nil, err
	}

	cctx, ok := txBoxed.(*Transaction)
	if !ok {
		return nil, errors.Errorf("received transaction of wrong type [%T]", cctx)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("received transaction with id [%s]", cctx.ID())
	}
	if !opt.NoTransactionVerification {
		// Check that the transaction is valid
		if err := cctx.IsValid(); err != nil {
			return nil, errors.WithMessagef(err, "invalid transaction %s", cctx.ID())
		}
	}

	return cctx, nil
}

// ID returns the ID of this transaction. It is equal to the underlying transaction's ID.
func (t *Transaction) ID() string {
	return t.Payload.ID
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

func (t *Transaction) Request() *token.Request {
	return t.Payload.TokenRequest
}

// Bytes returns the serialized version of the transaction.
// If eIDs is not nil, then metadata is filtered by the passed eIDs.
func (t *Transaction) Bytes(eIDs ...string) ([]byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("marshalling tx, id [%s], for EIDs [%x]", t.Payload.TxID.String(), eIDs)
	}
	return marshal(t, eIDs...)
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

func (t *Transaction) InputsAndOutputs() (*token.InputStream, *token.OutputStream, error) {
	return t.TokenRequest.InputsAndOutputs()
}

// IsValid checks that the transaction is well-formed.
// This means checking that the embedded TokenRequest is valid.
func (t *Transaction) IsValid() error {
	return t.TokenRequest.IsValid()
}

func (t *Transaction) MarshallToAudit() ([]byte, error) {
	return t.TokenRequest.MarshalToAudit()
}

// Selector returns the default token selector for this transaction
func (t *Transaction) Selector() (token.Selector, error) {
	return t.TokenService().SelectorManager().NewSelector(t.ID())
}

func (t *Transaction) Release() {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("releasing resources for tx [%s]", t.ID())
	}
	if err := t.TokenService().SelectorManager().Unlock(t.ID()); err != nil {
		logger.Warnf("failed releasing tokens locked by [%s], [%s]", t.ID(), err)
	}

	pub, err := publisher(t.SP)
	if err != nil {
		return
	}
	publishAbortTx(pub, t)
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
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Storing transient for [%s]", t.ID())
	}
	raw, err := t.TokenRequest.MetadataToBytes()
	if err != nil {
		return err
	}

	if err := t.Payload.Transient.Set(keys.TokenRequestMetadata, raw); err != nil {
		return err
	}

	return network.GetInstance(t.SP, t.Network(), t.Channel()).StoreTransient(t.ID(), t.Payload.Transient)
}

func (t *Transaction) setEnvelope(envelope *network.Envelope) error {
	if len(envelope.Nonce()) != 0 {
		networkTxID := &network.TxID{
			Nonce:   envelope.Nonce(),
			Creator: envelope.Creator(),
		}
		tempTXID := network.GetInstance(t.SP, t.Network(), t.Channel()).ComputeTxID(networkTxID)
		if tempTXID != envelope.TxID() {
			return errors.Errorf("txid mismatch, expected [%s], got [%s]", tempTXID, envelope.TxID())
		}
	}

	if t.Payload.ID != envelope.TxID() {
		return errors.Errorf("txid mismatch, expected [%s], got [%s]", t.Payload.ID, envelope.TxID())
	}
	t.Envelope = envelope

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("setting envelope [%s]", envelope.String())
	}
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

type TransactionSer struct {
	Nonce        []byte
	Creator      []byte
	ID           string
	Network      string
	Channel      string
	Namespace    string
	Signer       []byte
	Transient    []byte
	TokenRequest []byte
	Envelope     []byte
}

func marshal(t *Transaction, eIDs ...string) ([]byte, error) {
	var err error

	var transientRaw []byte
	if len(t.Payload.Transient) != 0 {
		transientRaw, err = MarshalMeta(t.Payload.Transient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal transient")
		}
	}

	var tokenRequestRaw []byte
	if t.Payload.TokenRequest != nil {
		req := t.Payload.TokenRequest
		// If eIDs are specified, we only marshal the metadata for the passed eIDs
		if len(eIDs) != 0 {
			req, err = t.Payload.TokenRequest.FilterMetadataBy(eIDs...)
			if err != nil {
				return nil, errors.Wrap(err, "failed to filter metadata")
			}
		}
		tokenRequestRaw, err = req.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal token request")
		}
	}

	var envRaw []byte
	if t.Payload.Envelope != nil {
		envRaw, err = t.Envelope.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal envelope")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction envelope [%s]", t.Envelope.String())
		}
	}

	res, err := asn1.Marshal(TransactionSer{
		Nonce:        t.Payload.TxID.Nonce,
		Creator:      t.Payload.TxID.Creator,
		ID:           t.Payload.ID,
		Network:      t.Payload.Network,
		Channel:      t.Payload.Channel,
		Namespace:    t.Payload.Namespace,
		Signer:       t.Payload.Signer,
		Transient:    transientRaw,
		TokenRequest: tokenRequestRaw,
		Envelope:     envRaw,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction")
	}
	return res, nil
}

func unmarshal(sp view2.ServiceProvider, p *Payload, raw []byte) error {
	var ser TransactionSer
	if _, err := asn1.Unmarshal(raw, &ser); err != nil {
		return errors.Wrapf(err, "failed unmarshalling transaction [%s]", string(raw))
	}

	p.TxID.Nonce = ser.Nonce
	p.TxID.Creator = ser.Creator
	p.ID = ser.ID
	p.Network = ser.Network
	p.Channel = ser.Channel
	p.Namespace = ser.Namespace
	p.Signer = ser.Signer
	p.Transient = make(map[string][]byte)
	if len(ser.Transient) != 0 {
		meta, err := UnmarshalMeta(ser.Transient)
		if err != nil {
			return errors.Wrap(err, "failed unmarshalling transient")
		}
		p.Transient = meta
	}
	if len(ser.TokenRequest) != 0 {
		if err := p.TokenRequest.FromBytes(ser.TokenRequest); err != nil {
			return errors.Wrap(err, "failed unmarshalling token request")
		}
	}
	if p.Envelope == nil {
		p.Envelope = network.GetInstance(sp, p.Network, p.Channel).NewEnvelope()
	}
	if len(ser.Envelope) != 0 {
		if err := p.Envelope.FromBytes(ser.Envelope); err != nil {
			return errors.Wrapf(err, "failed unmarshalling envelope [%d]", len(ser.Envelope))
		}
		// if err := t.setEnvelope(t.Envelope); err != nil {
		// 	return errors.Wrap(err, "failed setting envelope")
		// }
	}
	return nil
}
