/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	TokenNamespace = "tns"
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
	Envelope     *network.Envelope
}

type Transaction struct {
	*Payload

	TMS              *token.ManagementService
	NetworkProvider  GetNetworkFunc
	Opts             *TxOptions
	Context          context.Context
	FromRaw          []byte
	EndpointResolver *endpoint.Service
}

// NewAnonymousTransaction returns a new anonymous token transaction customized with the passed opts
func NewAnonymousTransaction(context view.Context, opts ...TxOption) (*Transaction, error) {
	txOpts, err := CompileOpts(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}
	tms := token.GetManagementService(
		context,
		token.WithTMSID(txOpts.TMSID),
	)
	net := network.GetInstance(context, tms.Network(), tms.Channel())
	if net == nil {
		return nil, errors.New("failed to get network")
	}
	id, err := net.AnonymousIdentity()
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting anonymous identity for transaction")
	}

	return NewTransaction(context, id, opts...)
}

// NewTransaction returns a new token transaction customized with the passed opts that will be signed by the passed signer.
// A valid signer is a signer that the target network recognizes as so. For example, in case of fabric, the signer must be a valid fabric identity.
// If the passed signer is nil, then the default identity is used.
func NewTransaction(context view.Context, signer view.Identity, opts ...TxOption) (*Transaction, error) {
	txOpts, err := CompileOpts(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed compiling tx options")
	}

	if txOpts.AnonymousTransaction && signer == nil {
		// set the signer to anonymous
		tms := token.GetManagementService(
			context,
			token.WithTMSID(txOpts.TMSID),
		)
		net := network.GetInstance(context, tms.Network(), tms.Channel())
		if net == nil {
			return nil, errors.New("failed to get network")
		}
		id, err := net.AnonymousIdentity()
		if err != nil {
			return nil, errors.WithMessage(err, "failed getting anonymous identity for transaction")
		}
		signer = id
	}

	tms := token.GetManagementService(
		context,
		token.WithTMSID(txOpts.TMSID),
	)
	networkService := network.GetInstance(context, tms.Network(), tms.Channel())
	networkProvider := network.GetProvider(context).GetNetwork

	var txID network.TxID
	if len(txOpts.NetworkTxID.Creator) != 0 {
		txID = txOpts.NetworkTxID
		signer = txID.Creator
	} else {
		if signer.IsNone() {
			signer = networkService.LocalMembership().DefaultIdentity()
		}
		txID = network.TxID{Creator: signer}
	}
	id := networkService.ComputeTxID(&txID)
	tr, err := tms.NewRequest(token.RequestAnchor(id))
	if err != nil {
		return nil, errors.WithMessage(err, "failed init token request")
	}

	tx := &Transaction{
		Payload: &Payload{
			Signer:       signer,
			TokenRequest: tr,
			Envelope:     nil,
			TxID:         txID,
			ID:           id,
			Network:      tms.Network(),
			Channel:      tms.Channel(),
			Namespace:    tms.Namespace(),
			Transient:    map[string][]byte{},
		},
		TMS:              tms,
		NetworkProvider:  networkProvider,
		Opts:             txOpts,
		Context:          context.Context(),
		EndpointResolver: endpoint.GetService(context),
	}
	context.OnError(tx.Release)
	return tx, nil
}

func NewTransactionFromBytes(context view.Context, raw []byte) (*Transaction, error) {
	tx := &Transaction{
		Payload: &Payload{
			Transient:    map[string][]byte{},
			TokenRequest: token.NewRequest(nil, ""),
		},
		Context: context.Context(),
		FromRaw: raw,
	}
	networkProvider := network.GetProvider(context).GetNetwork
	if err := unmarshal(networkProvider, tx.Payload, raw); err != nil {
		return nil, err
	}
	logger.DebugfContext(context.Context(), "unmarshalling tx, id [%s]", tx.TxID)
	tms := token.GetManagementService(context,
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(tx.Namespace()),
	)
	tx.TMS = tms
	tx.NetworkProvider = networkProvider
	tx.TokenRequest.SetTokenService(tms)
	if tx.ID() != string(tx.TokenRequest.ID()) {
		return nil, errors.Errorf("invalid transaction, transaction ids do not match [%s][%s]", tx.ID(), tx.TokenRequest.ID())
	}
	context.OnError(tx.Release)
	return tx, nil
}

func ReceiveTransaction(context view.Context, opts ...TxOption) (*Transaction, error) {
	opt, err := CompileOpts(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to parse options")
	}
	logger.DebugfContext(context.Context(), "receive a new transaction...")

	txBoxed, err := context.RunView(NewReceiveTransactionView(), view.WithSameContext())
	if err != nil {
		return nil, errors.WithMessage(err, "failed to receive transaction")
	}

	cctx, ok := txBoxed.(*Transaction)
	if !ok {
		return nil, errors.Errorf("received transaction of wrong type [%T]", cctx)
	}
	logger.DebugfContext(context.Context(), "received transaction with id [%s]", cctx.ID())
	if !opt.NoTransactionVerification {
		// Check that the transaction is valid
		if err := cctx.IsValid(context.Context()); err != nil {
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
	return t.TokenRequest
}

func (t *Transaction) NetworkTxID() network.TxID {
	return t.TxID
}

// Bytes returns the serialized version of the transaction.
// If eIDs is not nil, then metadata is filtered by the passed eIDs.
func (t *Transaction) Bytes(ctx context.Context, eIDs ...string) ([]byte, error) {
	logger.DebugfContext(ctx, "marshalling tx, id [%s], for EIDs [%x]", t.TxID, eIDs)
	return marshal(ctx, t, eIDs...)
}

// Issue appends a new Issue operation to the TokenRequest inside this transaction
func (t *Transaction) Issue(wallet *token.IssuerWallet, receiver view.Identity, typ token2.Type, q uint64, opts ...token.IssueOption) error {
	_, err := t.TokenRequest.Issue(t.Context, wallet, receiver, typ, q, opts...)
	return err
}

// Transfer appends a new Transfer operation to the TokenRequest inside this transaction
func (t *Transaction) Transfer(wallet *token.OwnerWallet, typ token2.Type, values []uint64, owners []view.Identity, opts ...token.TransferOption) error {
	_, err := t.TokenRequest.Transfer(t.Context, wallet, typ, values, owners, opts...)
	return err
}

func (t *Transaction) Redeem(wallet *token.OwnerWallet, typ token2.Type, value uint64, opts ...token.TransferOption) error {
	// build the redeem action
	action, err := t.TokenRequest.Redeem(t.Context, wallet, typ, value, opts...)
	if err != nil {
		return err
	}

	// check if the opts contain the issuer's network identity,
	// if yes, then bind it to the issuer identity in the redeem action

	// compile the options
	options, err := token.CompileTransferOptions(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to compile transfer options")
	}
	issuerNetworkIdentity, err := GetFSCIssuerIdentityFromOpts(options.Attributes)
	if err != nil {
		return errors.Wrap(err, "failed to get issuer identity")
	}
	if !issuerNetworkIdentity.IsNone() {
		if err := t.EndpointResolver.Bind(t.Context, issuerNetworkIdentity, action.GetIssuer()); err != nil {
			return errors.Wrapf(err, "failed to bind issuer identity [%s]", action.GetIssuer())
		}
	}
	return nil
}

// Upgrade performs an upgrade operation of the passed ledger tokens.
// A proof and its challenge will be used to verify that the request of upgrade is legit.
// If the proof verifies then the passed wallet will be used to issue a new amount of tokens
// matching those whose upgrade has been requested.
func (t *Transaction) Upgrade(
	wallet *token.IssuerWallet,
	receiver token.Identity,
	challenge token.TokensUpgradeChallenge,
	tokens []token2.LedgerToken,
	proof token.TokensUpgradeProof,
	opts ...token.IssueOption,
) error {
	_, err := t.TokenRequest.Upgrade(t.Context, wallet, receiver, challenge, tokens, proof, opts...)
	return err
}

func (t *Transaction) Outputs(ctx context.Context) (*token.OutputStream, error) {
	return t.TokenRequest.Outputs(ctx)
}

func (t *Transaction) Inputs(ctx context.Context) (*token.InputStream, error) {
	return t.TokenRequest.Inputs(ctx)
}

func (t *Transaction) InputsAndOutputs(ctx context.Context) (*token.InputStream, *token.OutputStream, map[string][]byte, error) {
	return t.TokenRequest.InputsAndOutputs(ctx)
}

// IsValid checks that the transaction is well-formed.
// This means checking that the embedded TokenRequest is valid.
func (t *Transaction) IsValid(ctx context.Context) error {
	return t.TokenRequest.IsValid(ctx)
}

func (t *Transaction) MarshallToAudit() ([]byte, error) {
	return t.TokenRequest.MarshalToAudit()
}

// Selector returns the default token selector for this transaction
func (t *Transaction) Selector() (token.Selector, error) {
	sm, err := t.TokenService().SelectorManager()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get selector manager")
	}
	return sm.NewSelector(t.ID())
}

func (t *Transaction) CloseSelector() error {
	sm, err := t.TokenService().SelectorManager()
	if err != nil {
		return errors.WithMessagef(err, "failed to get selector manager")
	}
	return sm.Close(t.ID())
}

func (t *Transaction) Release() {
	ctx := context.Background()
	logger.DebugfContext(ctx, "releasing resources for tx [%s]", t.ID())
	sm, err := t.TokenService().SelectorManager()
	if err != nil {
		logger.WarnfContext(ctx, "failed to get token selector [%s]", err)
	} else {
		if err := sm.Unlock(context.Background(), t.ID()); err != nil {
			logger.WarnfContext(ctx, "failed releasing tokens locked by [%s], [%s]", t.ID(), err)
		}
	}
}

func (t *Transaction) TokenService() *token.ManagementService {
	return t.TMS
}

func (t *Transaction) ApplicationMetadata(k string) []byte {
	return t.TokenRequest.ApplicationMetadata(k)
}

func (t *Transaction) SetApplicationMetadata(k string, v []byte) {
	t.TokenRequest.SetApplicationMetadata(k, v)
}

func (t *Transaction) TMSID() token.TMSID {
	return t.TokenRequest.TokenService.ID()
}

func (t *Transaction) appendPayload(payload *Payload) error {
	// TODO: change this
	t.TokenRequest = payload.TokenRequest
	t.Transient = payload.Transient
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
	// for _, issueMetadata := range payload.ValidationRecords.Issues {
	//	t.Payload.ValidationRecords.Issues = append(t.Payload.ValidationRecords.Issues, issueMetadata)
	// }
	// for _, transferMetadata := range payload.ValidationRecords.Transfers {
	//	t.Payload.ValidationRecords.Transfers = append(t.Payload.ValidationRecords.Transfers, transferMetadata)
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
