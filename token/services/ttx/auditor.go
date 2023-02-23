/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// TxStatus is the status of a transaction
type TxStatus = ttxdb.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown = ttxdb.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending TxStatus = ttxdb.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed TxStatus = ttxdb.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted TxStatus = ttxdb.Deleted
)

type txAuditor struct {
	sp                      view2.ServiceProvider
	w                       *token.AuditorWallet
	auditor                 *auditor.Auditor
	db                      *ttxdb.DB
	transactionInfoProvider *TransactionInfoProvider
}

func NewAuditor(sp view2.ServiceProvider, w *token.AuditorWallet) *txAuditor {
	return &txAuditor{
		sp:                      sp,
		w:                       w,
		auditor:                 auditor.New(sp, w),
		db:                      ttxdb.Get(sp, w),
		transactionInfoProvider: NewTransactionInfoProvider(sp, w.TMS()),
	}
}

func (a *txAuditor) Validate(tx *Transaction) error {
	return a.auditor.Validate(tx.TokenRequest)
}

func (a *txAuditor) Audit(tx *Transaction) (*token.InputStream, *token.OutputStream, error) {
	return a.auditor.Audit(tx)
}

// Release unlocks the passed enrollment IDs.
func (a *txAuditor) Release(tx *Transaction) {
	a.auditor.Release(tx)
}

// NewQueryExecutor returns a new query executor. The query executor is used to
// execute queries against the auditor's DB.
// The function `Done` on the query executor must be called when it is no longer needed.
func (a *txAuditor) NewQueryExecutor() *auditor.QueryExecutor {
	return a.auditor.NewQueryExecutor()
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *txAuditor) SetStatus(txID string, status TxStatus) error {
	return a.db.SetStatus(txID, status)
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *txAuditor) TransactionInfo(txID string) (*TransactionInfo, error) {
	return a.transactionInfoProvider.TransactionInfo(txID)
}

type RegisterAuditorView struct {
	TMSID     token.TMSID
	AuditView view.View
}

func NewRegisterAuditorView(auditView view.View, opts ...token.ServiceOption) *RegisterAuditorView {
	options, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil
	}
	return &RegisterAuditorView{
		AuditView: auditView,
		TMSID:     options.TMSID(),
	}
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	// register responder
	if err := view2.GetRegistry(context).RegisterResponder(r.AuditView, &AuditingViewInitiator{}); err != nil {
		return nil, errors.Wrapf(err, "failed to register auditor view")
	}
	// enable processing of all token transactions for the given network and namespace
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	if tms == nil {
		return nil, errors.Errorf("cannot find tms for [%s]", r.TMSID)
	}
	net := network.GetInstance(context, tms.Network(), tms.Channel())
	if tms == nil {
		return nil, errors.Errorf("cannot find network for [%s]", tms.ID())
	}
	if err := net.ProcessNamespace(tms.Namespace()); err != nil {
		return nil, errors.WithMessagef(err, "failed to register namespace for processing [%s]", tms.Network())
	}
	return nil, nil
}

type AuditingViewInitiator struct {
	tx    *Transaction
	local bool
}

func newAuditingViewInitiator(tx *Transaction, local bool) *AuditingViewInitiator {
	return &AuditingViewInitiator{tx: tx, local: local}
}

func (a *AuditingViewInitiator) Call(context view.Context) (interface{}, error) {
	var err error
	var session view.Session
	if a.local {
		session, err = a.startLocal(context)
	} else {
		session, err = a.startRemote(context)
	}
	if err != nil {
		return nil, errors.WithMessage(err, "failed starting auditing session")
	}

	timeout := time.NewTimer(time.Minute)
	defer timeout.Stop()

	// Receive signature
	logger.Debugf("Receiving signature for [%s]", a.tx.ID())
	ch := session.Receive()
	var msg *view.Message
	select {
	case msg = <-ch:
		logger.Debugf("reply received from %s", a.tx.Opts.Auditor)
	case <-timeout.C:
		return nil, errors.Errorf("Timeout from party %s", a.tx.Opts.Auditor)
	}
	if msg.Status == view.ERROR {
		return nil, errors.New(string(msg.Payload))
	}

	// Check signature
	signed, err := a.tx.MarshallToAudit()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling message to sign")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Verifying auditor signature on [%s][%s][%s]", a.tx.Opts.Auditor.UniqueID(), hash.Hashable(signed).String(), a.tx.ID())
	}

	validAuditing := false
	for _, auditor := range a.tx.TokenService().PublicParametersManager().Auditors() {
		v, err := a.tx.TokenService().SigService().AuditorVerifier(auditor)
		if err != nil {
			logger.Debugf("Failed to get auditor verifier for %s", auditor.UniqueID())
			continue
		}
		if err := v.Verify(signed, msg.Payload); err != nil {
			logger.Debugf("Failed verifying auditor signature [%s][%s]", hash.Hashable(signed).String(), a.tx.TokenRequest.Anchor)
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Auditor signature verified [%s][%s]", auditor, base64.StdEncoding.EncodeToString(msg.Payload))
			}
			validAuditing = true
			break
		}
	}
	if !validAuditing {
		return nil, errors.Errorf("failed verifying auditor signature [%s][%s]", hash.Hashable(signed).String(), a.tx.TokenRequest.Anchor)
	}
	a.tx.TokenRequest.AddAuditorSignature(msg.Payload)

	logger.Debug("Auditor signature verified")
	return session, nil
}

func (a *AuditingViewInitiator) startRemote(context view.Context) (view.Session, error) {
	logger.Debugf("Starting remote auditing session with [%s] for [%s]", a.tx.Opts.Auditor.UniqueID(), a.tx.ID())
	session, err := context.GetSession(a, a.tx.Opts.Auditor)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting session")
	}

	// Send transaction
	txRaw, err := a.tx.Bytes()
	if err != nil {
		return nil, err
	}
	err = session.Send(txRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction")
	}

	return session, nil
}

func (a *AuditingViewInitiator) startLocal(context view.Context) (view.Session, error) {
	logger.Debugf("Starting local auditing for %s", a.tx.ID())

	// This code is executed everytime the auditor is the same as the
	// initiator of a token transaction.
	// For example, if an issuer is also an auditor, then when the issuer asks
	// for auditing, the issuer is essentially talking to itself.
	// FSC does not yet support opening communication session to itself,
	// therefore we create a fake bidirectional communication channel between
	// the AuditingViewInitiator view and its registered responder.
	// Notice also the use of view.AsResponder(right) to run the responder
	// using a predefined session.
	// This code can be removed once FSC supports opening communication session to self.

	// Prepare a bidirectional channel
	// Give to the responder view the right channel, and keep for
	// AuditingViewInitiator the left channel.
	biChannel, err := NewLocalBidirectionalChannel("", context.ID(), "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating session")
	}
	left := biChannel.LeftSession()
	right := biChannel.RightSession()

	// Send transaction
	txRaw, err := a.tx.Bytes()
	if err != nil {
		return nil, err
	}
	err = left.Send(txRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction")
	}

	// execute the auditor responder using the fake communication session
	responderView, err := view2.GetRegistry(context).GetResponder(&AuditingViewInitiator{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get auditor view")
	}
	// Run the view in a new goroutine
	RunView(context, responderView, view.AsResponder(right))

	return left, nil
}

type AuditApproveView struct {
	w  *token.AuditorWallet
	tx *Transaction
}

func NewAuditApproveView(w *token.AuditorWallet, tx *Transaction) *AuditApproveView {
	return &AuditApproveView{w: w, tx: tx}
}

func (a *AuditApproveView) Call(context view.Context) (interface{}, error) {
	// Append audit records
	if err := auditor.New(context, a.w).Append(a.tx); err != nil {
		return nil, errors.Wrapf(err, "failed appending audit records for transaction %s", a.tx.ID())
	}

	if err := a.signAndSendBack(context); err != nil {
		return nil, err
	}

	labels := []string{
		"network", a.tx.Network(),
		"channel", a.tx.Channel(),
		"namespace", a.tx.Namespace(),
	}
	GetMetrics(context).AuditApprovedTransactions.With(labels...).Add(1)
	return nil, nil
}

func (a *AuditApproveView) signAndSendBack(context view.Context) error {
	logger.Debugf("Signing and sending back transaction... [%s]", a.tx.ID())
	// Sign
	aid, err := a.w.GetAuditorIdentity()
	if err != nil {
		return errors.WithMessagef(err, "failed getting auditor identity for [%s]", context.Me())
	}
	signer, err := a.w.GetSigner(aid)
	if err != nil {
		return errors.WithMessagef(err, "failed getting signing identity for auditor identity [%s]", context.Me())
	}

	err = a.tx.storeTransient()
	if err != nil {
		return errors.Wrapf(err, "failed storing transient for [%s]", a.tx.ID())
	}

	logger.Debug("signer at auditor", signer, aid)

	raw, err := a.tx.MarshallToAudit()
	if err != nil {
		return errors.Wrapf(err, "failed marshalling tx [%s] to audit", a.tx.ID())
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Audit Approve [%s][%s][%s]", aid.UniqueID(), hash.Hashable(raw).String(), a.tx.TokenRequest.Anchor)
	}
	sigma, err := signer.Sign(raw)
	if err != nil {
		return errors.Wrapf(err, "failed sign audit message for tx [%s]", a.tx.ID())
	}
	logger.Debug("auditor sending sigma back", hash.Hashable(sigma))
	session := context.Session()
	if err := session.Send(sigma); err != nil {
		return errors.WithMessagef(err, "failed sending back auditor signature")
	}

	logger.Debugf("Signing and sending back transaction...done [%s]", a.tx.ID())

	if err := a.waitEnvelope(context); err != nil {
		return errors.WithMessagef(err, "failed obtaining auditor signature")
	}
	return nil
}

func (a *AuditApproveView) waitEnvelope(context view.Context) error {
	logger.Debugf("Waiting for envelope... [%s]", a.tx.ID())
	tx, err := ReceiveTransaction(context, WithNoTransactionVerification())
	if err != nil {
		return errors.Wrapf(err, "failed to receive transaction with network envelope")
	}
	logger.Debugf("Waiting for envelope...transaction received[%s]", a.tx.ID())

	// Processes
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Processes envelope...")
	}
	if tx.Payload == nil {
		return errors.Errorf("expected transaction payload not found")
	}
	env := tx.Payload.Envelope
	if env == nil {
		return errors.Errorf("expected envelope not found")
	}
	// Ack for distribution
	// Send the signature back
	rawRequest, err := tx.Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed marshalling tx [%s]", tx.ID())
	}

	backend := network.GetInstance(context, tx.Network(), tx.Channel())
	rws, err := backend.GetRWSet(tx.ID(), env.Results())
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset for tx [%s]", tx.ID())
	}
	rws.Done()

	rawEnv, err := env.Bytes()
	if err != nil {
		return errors.WithMessagef(err, "failed marshalling tx env [%s]", tx.ID())
	}
	if err := backend.StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return errors.WithMessagef(err, "failed storing tx env [%s]", tx.ID())
	}

	var sigma []byte
	logger.Debugf("signing ack response: %s", hash.Hashable(rawRequest))
	signer, err := view2.GetSigService(context).GetSigner(view2.GetIdentityProvider(context).DefaultIdentity())
	if err != nil {
		return errors.WithMessagef(err, "failed getting signing identity for [%s]", view2.GetIdentityProvider(context).DefaultIdentity())
	}
	sigma, err = signer.Sign(rawRequest)
	if err != nil {
		return errors.WithMessage(err, "failed to sign ack response")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("ack response: [%s] from [%s]", hash.Hashable(sigma), view2.GetIdentityProvider(context).DefaultIdentity())
	}
	session := context.Session()
	if err := session.Send(sigma); err != nil {
		return errors.WithMessage(err, "failed sending ack")
	}

	logger.Debugf("Waiting for envelope...done [%s]", a.tx.ID())

	return nil
}
