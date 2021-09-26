/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"bytes"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc"

	"github.com/pkg/errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb"
)

type txAuditor struct {
	auditor *auditor.Auditor
}

func NewAuditor(sp view2.ServiceProvider, w *token.AuditorWallet) *txAuditor {
	return &txAuditor{
		auditor: auditor.New(sp, w),
	}
}

func (a *txAuditor) Validate(tx *Transaction) error {
	return a.auditor.Validate(tx.TokenRequest)
}

func (a *txAuditor) Audit(tx *Transaction) (*token.InputStream, *token.OutputStream, error) {
	return a.auditor.Audit(tx.TokenRequest)
}

func (a *txAuditor) NewQueryExecutor() *auditor.QueryExecutor {
	return a.auditor.NewQueryExecutor()
}

type RegisterAuditorView struct {
	TMSID     token.TMSID
	Id        view.Identity
	AuditView view.View
}

func NewRegisterAuditorView(id view.Identity, auditView view.View) *RegisterAuditorView {
	return &RegisterAuditorView{Id: id, AuditView: auditView}
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	view2.GetRegistry(context).RegisterResponder(r.AuditView, &AuditingViewInitiator{})

	return context.RunView(tcc.NewRegisterAuditorView(r.TMSID, r.Id))
}

func NewCollectAuditorEndorsement(tx *Transaction) *AuditingViewInitiator {
	return newAuditingViewInitiator(tx)
}

type AuditingViewInitiator struct {
	tx *Transaction
}

func newAuditingViewInitiator(tx *Transaction) *AuditingViewInitiator {
	return &AuditingViewInitiator{tx: tx}
}

func (a *AuditingViewInitiator) Call(context view.Context) (interface{}, error) {
	auditor := a.tx.opts.auditor
	if auditor.IsNone() {
		logger.Warnf("no auditor specified, skipping")

		return nil, nil
	}

	session, err := context.GetSession(a, auditor)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting session")
	}
	// Wait to receive a content back
	ch := session.Receive()

	txRaw, err := a.tx.Bytes()
	if err != nil {
		return nil, err
	}
	err = session.Send(txRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction")
	}

	// Wait for the answer
	var msg *view.Message
	select {
	case msg = <-ch:
		logger.Debugf("collect auditor signature: reply received from %s", auditor)
	case <-time.After(60 * time.Second):
		return nil, errors.Errorf("Timeout from party %s", auditor)
	}
	if msg.Status == view.ERROR {
		return nil, errors.New(string(msg.Payload))
	}

	// The response contains a  marshalled ProposalResponse message
	proposalResponse, err := fabric.GetFabricNetworkService(context, a.tx.Network()).TransactionManager().NewProposalResponseFromBytes(msg.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling received proposal response")
	}
	endorser := view.Identity(proposalResponse.Endorser())

	// Verify signatures
	res, err := a.tx.Results()
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting tx results")
	}
	verifier, err := a.tx.TokenService().SigService().AuditorVerifier(endorser)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting verifier for party %s", auditor.String())
	}
	err = verifier.Verify(append(proposalResponse.Payload(), endorser...), proposalResponse.EndorserSignature())
	if err != nil {
		return nil, errors.Wrapf(err, "failed verifying endorsement for party %s", endorser.String())
	}
	// Now results can be equal to what this node has proposed or different
	if !bytes.Equal(res, proposalResponse.Results()) {
		return nil, errors.Errorf("received different results")
	}

	err = a.tx.AppendProposalResponse(proposalResponse)
	if err != nil {
		return nil, errors.Wrap(err, "failed appending received proposal response")
	}

	return nil, nil
}

type AuditApproveView struct {
	w  *token.AuditorWallet
	tx *Transaction
}

func NewAuditApproveView(w *token.AuditorWallet, tx *Transaction) *AuditApproveView {
	return &AuditApproveView{w: w, tx: tx}
}

func (a *AuditApproveView) Call(context view.Context) (interface{}, error) {
	aid, err := a.w.GetAuditorIdentity()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting auditor identity for [%s]", context.Me())
	}
	signer, err := a.w.GetSigner(aid)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting signing identity for auditor identity [%s]", context.Me())
	}

	logger.Debugf("store audit records...")
	auditRecord, err := a.tx.TokenRequest.AuditRecord()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting audit records for tx [%s]", a.tx.ID())
	}
	if err := auditdb.GetAuditDB(context, a.w).Append(auditRecord); err != nil {
		return nil, errors.WithMessagef(err, "failed appening audit records for tx [%s]", a.tx.ID())
	}
	logger.Debugf("store audit records...done")

	if err := a.tx.EndorseWithSigner(aid, signer); err != nil {
		return nil, errors.Wrapf(err, "failed marshalling tx [%s] to audit", a.tx.ID())
	}

	// store transaction
	txRaw, err := a.tx.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling tx")
	}
	ch, err := fabric.GetFabricNetworkService(context, a.tx.Network()).Channel(a.tx.Channel())
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting channel [%s:%s]", a.tx.Network(), a.tx.Channel())
	}
	if err := ch.Vault().StoreTransaction(a.tx.ID(), txRaw); err != nil {
		return nil, errors.WithMessagef(err, "failed storing tx env [%s]", a.tx.ID())
	}

	// send reply
	raw, err := a.tx.ProposalResponse()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling response")
	}

	session := context.Session()
	if err := session.Send(raw); err != nil {
		return nil, errors.WithMessagef(err, "failed sending back auditor signature")
	}

	return nil, nil
}
