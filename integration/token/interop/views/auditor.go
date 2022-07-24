/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed receiving transaction")

	assert.NoError(tx.IsValid(), "failed verifying transaction")

	w := ttx.MyAuditorWallet(context, token.WithTMSID(tx.TokenService().ID()))
	assert.NotNil(w, "failed getting default auditor wallet")
	auditor := ttx.NewAuditor(context, w)
	assert.NoError(auditor.Validate(tx), "failed auditing verification")

	_, _, err = auditor.Audit(tx)
	assert.NoError(err, "failed retrieving inputs and outputs")

	return context.RunView(ttx.NewAuditApproveView(w, tx))
}

type RegisterAuditor struct {
	TMSID token.TMSID
}

type RegisterAuditorView struct {
	*RegisterAuditor
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	return context.RunView(ttx.NewRegisterAuditorView(
		&AuditView{},
		token.WithTMSID(r.TMSID),
	))
}

type RegisterAuditorViewFactory struct{}

func (p *RegisterAuditorViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterAuditorView{RegisterAuditor: &RegisterAuditor{}}
	err := json.Unmarshal(in, f.RegisterAuditor)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
