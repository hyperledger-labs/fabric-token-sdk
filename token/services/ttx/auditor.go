/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	view3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/view"
)

type TxAuditor struct {
	w                       *token.AuditorWallet
	auditor                 *auditor.Service
	auditDB                 *auditdb.StoreService
	transactionInfoProvider *TransactionInfoProvider
}

func NewAuditor(sp token.ServiceProvider, w *token.AuditorWallet) (*TxAuditor, error) {
	backend := auditor.New(sp, w)
	auditDB, err := auditdb.GetByTMSId(sp, w.TMS().ID())
	if err != nil {
		return nil, err
	}
	ttxDB, err := ttxdb.GetByTMSId(sp, w.TMS().ID())
	if err != nil {
		return nil, err
	}
	return NewTxAuditor(w, backend, auditDB, ttxDB), nil
}

func NewTxAuditor(w *token.AuditorWallet, backend *auditor.Service, auditDB *auditdb.StoreService, ttxDB *ttxdb.StoreService) *TxAuditor {
	return &TxAuditor{
		w:                       w,
		auditor:                 backend,
		auditDB:                 auditDB,
		transactionInfoProvider: newTransactionInfoProvider(w.TMS(), ttxDB),
	}
}

func (a *TxAuditor) Validate(tx *Transaction) error {
	return a.auditor.Validate(tx.Context, tx.TokenRequest)
}

func (a *TxAuditor) Audit(ctx context.Context, tx *Transaction) (*token.InputStream, *token.OutputStream, error) {
	return a.auditor.Audit(ctx, tx)
}

// Release unlocks the passed enrollment IDs.
func (a *TxAuditor) Release(ctx context.Context, tx *Transaction) {
	a.auditor.Release(ctx, tx)
}

// Transactions returns an iterator of transaction records filtered by the given params.
func (a *TxAuditor) Transactions(ctx context.Context, params QueryTransactionsParams, pagination Pagination) (*PageTransactionsIterator, error) {
	return a.auditDB.Transactions(ctx, params, pagination)
}

// NewPaymentsFilter returns a programmable filter over the payments sent or received by enrollment IDs.
func (a *TxAuditor) NewPaymentsFilter() *auditdb.PaymentsFilter {
	return a.auditDB.NewPaymentsFilter()
}

// NewHoldingsFilter returns a programmable filter over the holdings owned by enrollment IDs.
func (a *TxAuditor) NewHoldingsFilter() *auditdb.HoldingsFilter {
	return a.auditDB.NewHoldingsFilter()
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *TxAuditor) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) error {
	return a.auditDB.SetStatus(ctx, txID, status, message)
}

func (a *TxAuditor) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	return a.auditor.GetTokenRequest(ctx, txID)
}

func (a *TxAuditor) Check(ctx context.Context) ([]string, error) {
	return a.auditor.Check(ctx)
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
	return nil, nil
}

type AuditingViewInitiator struct {
	tx                               *Transaction
	local                            bool
	skipAuditorSignatureVerification bool
}

func newAuditingViewInitiator(tx *Transaction, local, skipAuditorSignatureVerification bool) *AuditingViewInitiator {
	return &AuditingViewInitiator{tx: tx, local: local, skipAuditorSignatureVerification: skipAuditorSignatureVerification}
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
		return nil, errors.WithMessagef(err, "failed starting auditing session")
	}

	// Receive signature
	logger.DebugfContext(context.Context(), "Receiving signature for [%s]", a.tx.ID())

	jsonSession := session2.NewFromSession(context, session)
	signature, err := jsonSession.ReceiveRawWithTimeout(time.Minute)
	if err != nil {
		logger.ErrorfContext(context.Context(), "failed to read audit event: %s", err)
		return nil, errors.WithMessagef(err, "failed to read audit event")
	}
	logger.DebugfContext(context.Context(), "reply received from %s", a.tx.Opts.Auditor)

	auditorIdentity, err := a.verifyAuditorSignature(context, signature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed verifying auditor signature")
	}

	a.tx.TokenRequest.AddAuditorSignature(auditorIdentity, signature)

	logger.Debug("auditor signature verified")
	return session, nil
}

func (a *AuditingViewInitiator) startRemote(context view.Context) (view.Session, error) {
	logger.DebugfContext(context.Context(), "Starting remote auditing session with [%s] for [%s]", a.tx.Opts.Auditor.UniqueID(), a.tx.ID())
	session, err := context.GetSession(a, a.tx.Opts.Auditor)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting session")
	}

	// Send transaction
	txRaw, err := a.tx.Bytes()
	if err != nil {
		return nil, err
	}
	err = session.SendWithContext(context.Context(), txRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction")
	}

	return session, nil
}

func (a *AuditingViewInitiator) startLocal(context view.Context) (view.Session, error) {
	logger.DebugfContext(context.Context(), "Starting local auditing for %s", a.tx.ID())

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
	biChannel, err := NewLocalBidirectionalChannel(context.Context(), "", context.ID(), "", nil)
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
	err = left.SendWithContext(context.Context(), txRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction")
	}

	// execute the auditor responder using the fake communication session
	responderView, err := view2.GetRegistry(context).GetResponder(&AuditingViewInitiator{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get auditor view")
	}
	// Run the view in a new goroutine
	view3.RunView(logger, context, responderView, view.AsResponder(right))

	return left, nil
}

func (a *AuditingViewInitiator) verifyAuditorSignature(context view.Context, signature []byte) (token.Identity, error) {
	logger.DebugfContext(context.Context(), "Validate auditing")

	if a.skipAuditorSignatureVerification {
		return a.tx.Opts.Auditor, nil
	}

	// check the signature
	signed, err := a.tx.MarshallToAudit()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling message to sign")
	}
	logger.DebugfContext(context.Context(), "Verifying auditor signature on [%s][%s][%s]", a.tx.Opts.Auditor, utils.Hashable(signed), a.tx.ID())

	for _, auditorID := range a.tx.TokenService().PublicParametersManager().PublicParameters().Auditors() {
		v, err := a.tx.TokenService().SigService().AuditorVerifier(context.Context(), auditorID)
		if err != nil {
			logger.DebugfContext(context.Context(), "failed to get auditor verifier for [%s]", auditorID)
			continue
		}
		logger.DebugfContext(context.Context(), "Verify auditor signature")
		if err := v.Verify(signed, signature); err != nil {
			logger.ErrorfContext(context.Context(), "failed verifying auditor signature [%s][%s][%s]", auditorID, utils.Hashable(signed), a.tx.TokenRequest.Anchor)
		} else {
			logger.DebugfContext(context.Context(), "auditor signature verified [%s][%s][%s]", auditorID, base64.StdEncoding.EncodeToString(signature), utils.Hashable(signed))
			return auditorID, nil
		}
	}
	return nil, errors.Errorf("failed verifying auditor signature [%s][%s]", utils.Hashable(signed).String(), a.tx.TokenRequest.Anchor)
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
	if err := auditor.New(context, a.w).Append(context.Context(), a.tx); err != nil {
		return nil, errors.Wrapf(err, "failed appending audit records for transaction %s", a.tx.ID())
	}

	if err := a.signAndSendBack(context); err != nil {
		return nil, err
	}

	// cache the token request into the tokens db
	t, err := tokens.GetService(context, a.tx.TMSID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens db for [%s]", a.tx.TMSID())
	}
	if err := t.CacheRequest(context.Context(), a.tx.TMSID(), a.tx.TokenRequest); err != nil {
		logger.Warnf("failed to cache token request [%s], this might cause delay, investigate when possible: [%s]", a.tx.TokenRequest.Anchor, err)
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
	logger.DebugfContext(context.Context(), "Signing and sending back transaction... [%s]", a.tx.ID())
	// Sign
	aid, err := a.w.GetAuditorIdentity()
	if err != nil {
		return errors.WithMessagef(err, "failed getting auditor identity for node [%s]", context.Me())
	}
	signer, err := a.w.GetSigner(context.Context(), aid)
	if err != nil {
		return errors.WithMessagef(err, "failed getting signing identity for auditor identity [%s]", aid)
	}

	raw, err := a.tx.MarshallToAudit()
	if err != nil {
		return errors.Wrapf(err, "failed marshalling tx [%s] to audit", a.tx.ID())
	}

	logger.DebugfContext(context.Context(), "Audit Approve [%s][%s][%s]", aid, utils.Hashable(raw), a.tx.TokenRequest.Anchor)
	sigma, err := signer.Sign(raw)
	if err != nil {
		return errors.Wrapf(err, "failed sign audit message for tx [%s]", a.tx.ID())
	}

	logger.DebugfContext(context.Context(), "auditor sending sigma back", utils.Hashable(sigma))
	if err := context.Session().SendWithContext(context.Context(), sigma); err != nil {
		return errors.WithMessagef(err, "failed sending back auditor signature")
	}
	logger.DebugfContext(context.Context(), "Signing and sending back transaction...done [%s]", a.tx.ID())

	return nil
}
