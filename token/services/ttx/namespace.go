/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
)

var logger = flogging.MustGetLogger("token-sdk.zkat")

type Namespace struct {
	tx           *endorser.Transaction
	opts         *txOptions
	TokenRequest *token.Request `json:"-"`
}

func NewNamespace(tx *endorser.Transaction, opts ...TxOption) (*Namespace, error) {
	txOpts, err := compile(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}

	n := &Namespace{
		tx:   tx,
		opts: txOpts,
	}
	if err := n.populate(); err != nil {
		return nil, errors.Wrapf(err, "failed populating namespace")
	}
	return n, nil
}

func (t *Namespace) Issue(wallet *token.IssuerWallet, receiver view.Identity, typ string, q uint64) error {
	action, err := t.TokenRequest.Issue(wallet, receiver, typ, q)
	if err != nil {
		return errors.Wrapf(err, "failed issuing")
	}
	return t.updateRWSetAndMetadata(action)
}

func (t *Namespace) Transfer(wallet *token.OwnerWallet, typ string, values []uint64, owners []view.Identity, opts ...token.TransferOption) error {
	action, err := t.TokenRequest.Transfer(wallet, typ, values, owners, opts...)
	if err != nil {
		return errors.Wrapf(err, "failed issuing")
	}
	return t.updateRWSetAndMetadata(action)
}

func (t *Namespace) Outputs() (*token.OutputStream, error) {
	return t.TokenRequest.Outputs()
}

func (t *Namespace) Inputs() (*token.InputStream, error) {
	return t.TokenRequest.Inputs()
}

func (t *Namespace) Verify() error {
	return t.TokenRequest.Verify()
}

func (t *Namespace) IsValid() error {
	return t.TokenRequest.IsValid()
}

func (t *Namespace) Signers() []view.Identity {
	var ids []view.Identity
	for _, issue := range t.TokenRequest.Issues() {
		ids = append(ids, issue.Issuer)
	}
	for _, transfer := range t.TokenRequest.Transfers() {
		for _, sender := range transfer.Senders {
			if t.tokenService().WalletManager().OwnerWalletByIdentity(sender) != nil {
				ids = append(ids, sender)
			}
		}
	}
	return ids
}

func (t *Namespace) Receivers() []view.Identity {
	var ids []view.Identity
	for _, issue := range t.TokenRequest.Issues() {
		for _, receiver := range issue.Receivers {
			if t.tokenService().WalletManager().OwnerWalletByIdentity(receiver) != nil {
				ids = append(ids, receiver)
			}
		}
	}
	for _, transfer := range t.TokenRequest.Transfers() {
		for _, receiver := range transfer.Receivers {
			if t.tokenService().WalletManager().OwnerWalletByIdentity(receiver) != nil {
				ids = append(ids, receiver)
			}
		}
	}
	return ids
}

func (t *Namespace) Endorsers() []view.Identity {
	var ids []view.Identity
	for _, issue := range t.TokenRequest.Issues() {
		ids = append(ids, issue.Issuer)
	}
	for _, transfer := range t.TokenRequest.Transfers() {
		ids = append(ids, transfer.Senders...)
	}
	for _, issue := range t.TokenRequest.Issues() {
		ids = append(ids, issue.Receivers...)
	}
	for _, transfer := range t.TokenRequest.Transfers() {
		ids = append(ids, transfer.Receivers...)
	}
	return ids
}

func (t *Namespace) SetProposal() {
	t.tx.SetProposal(t.tokenService().Namespace(), "Version-0.0", "")
}

func (t *Namespace) updateRWSetAndMetadata(action interface{}) error {
	rws, err := t.tx.RWSet()
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset")
	}

	// store token request in the rwset
	ns := t.tokenService().Namespace()
	key, err := keys.CreateTokenRequestKey(t.tx.ID())
	if err != nil {
		return errors.WithMessagef(err, "failed computing token request key")
	}
	tr, err := rws.GetState(ns, key)
	if err != nil {
		return errors.Wrapf(err, "failed to write token request [%s]", t.tx.ID())
	}
	if tr != nil {
		return errors.Wrapf(errors.New("token request with same ID already exists"), "failed to write token request [%s]", t.tx.ID())
	}
	tokenRequestRaw, err := t.TokenRequest.RequestToBytes()
	if err != nil {
		return errors.Wrapf(err, "failed marshalling token request [%s]", t.tx.ID())
	}
	logger.Debugf("Store Token Request in RWS [%s][%s]", t.tx.ID(), string(tokenRequestRaw))
	err = rws.SetState(ns, key, tokenRequestRaw)
	if err != nil {
		return errors.Wrapf(err, "failed to write token request [%s]", t.tx.ID())
	}

	// store metadata
	tokenRequestMetaRaw, err := t.TokenRequest.MetadataToBytes()
	if err != nil {
		return err
	}
	if err := t.tx.SetTransient("zkat", tokenRequestMetaRaw); err != nil {
		return errors.Wrapf(err, "failed storing metadata in transaction [%s]", t.tx.ID())
	}

	// commit action, if any
	if action != nil {
		issuingValidator := &allIssuersValid{}
		w := translator.New(issuingValidator, t.tx.ID(), rws, ns)
		err = w.Write(action)
		if err != nil {
			return errors.Wrap(err, "failed to write token action")
		}
		err = w.CommitTokenRequest(tokenRequestRaw)
		if err != nil {
			return errors.Wrap(err, "failed to write token request")
		}
	}

	return nil
}

func (t *Namespace) populate() error {
	rws, err := t.tx.RWSet()
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset")
	}

	key, err := keys.CreateTokenRequestKey(t.tx.ID())
	if err != nil {
		return errors.WithMessagef(err, "failed computing token request key")
	}
	requestRaw, err := rws.GetState(t.tokenService().Namespace(), key, fabric.FromIntermediate)
	if err != nil {
		return errors.WithMessagef(err, "failed computing token request key")
	}
	if len(requestRaw) == 0 {
		t.TokenRequest, err = t.tokenService().NewRequest(t.tx.ID())
		if err != nil {
			return errors.Wrapf(err, "failed creating new token request for transaction [%s]", t.tx.ID())
		}
		return nil
	}

	var metaRaw []byte
	if t.tx.ExistsTransientState("zkat") {
		metaRaw = t.tx.GetTransient("zkat")
		if len(metaRaw) == 0 {
			return errors.Errorf("failed loading metadata from transaction [%s], empty", t.tx.ID())
		}
	} else {
		logger.Warnf("transient metadata not found in tx [%d]", t.tx.ID())
	}

	logger.Debugf("Loaded Token Request from RWS [%s][%s]", t.tx.ID(), string(requestRaw))
	t.TokenRequest, err = t.tokenService().NewRequestFromBytes(t.tx.ID(), requestRaw, metaRaw)
	if err != nil {
		return errors.Wrapf(err, "failed unmarshalling request for transaction [%s]\n[%s]\n[%s]", t.tx.ID(), string(requestRaw), string(metaRaw))
	}
	return nil
}

func (t *Namespace) tokenService() *token.ManagementService {
	return token.GetManagementService(
		t.tx.ServiceProvider,
		token.WithNetwork(t.tx.Network()),
		token.WithChannel(t.tx.Channel()),
	)
}

func (t *Namespace) append(TokenRequest *token.Request, rwsRaw []byte) error {
	// append token request and meta
	if err := t.TokenRequest.Import(TokenRequest); err != nil {
		return errors.Wrapf(err, "failed importing request")
	}

	// append rws
	rws, err := t.tx.RWSet()
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset")
	}
	if err := rws.AppendRWSet(rwsRaw, t.tokenService().Namespace()); err != nil {
		return errors.WithMessagef(err, "failed getting rwset")
	}

	return t.updateRWSetAndMetadata(nil)
}
