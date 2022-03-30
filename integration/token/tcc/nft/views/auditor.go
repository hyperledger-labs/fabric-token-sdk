/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	tx, err := ttxcc.ReceiveTransaction(context)
	assert.NoError(err, "failed receiving transaction")

	w := ttxcc.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	// Validate
	auditor := ttxcc.NewAuditor(context, w)
	assert.NoError(auditor.Validate(tx), "failed auditing verification")

	return context.RunView(ttxcc.NewAuditApproveView(w, tx))
}

type RegisterAuditorView struct{}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	return context.RunView(ttxcc.NewRegisterAuditorView(
		&AuditView{},
	))
}

type RegisterAuditorViewFactory struct{}

func (p *RegisterAuditorViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterAuditorView{}
	return f, nil
}
