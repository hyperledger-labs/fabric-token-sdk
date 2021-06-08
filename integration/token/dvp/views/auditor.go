/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed receiving transaction")

	assert.NoError(tx.IsValid(), "failed verifying transaction")

	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")
	assert.NoError(ttx.NewAuditor(context, w).Validate(tx), "failed auditing verification")

	return context.RunView(ttx.NewAuditApproveView(w, tx))
}

type RegisterAuditorView struct {
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	return context.RunView(ttx.NewRegisterAuditorView(context.Me(), &AuditView{}))
}

type RegisterAuditorViewFactory struct{}

func (p *RegisterAuditorViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterAuditorView{}
	return f, nil
}
