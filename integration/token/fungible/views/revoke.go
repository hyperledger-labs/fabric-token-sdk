/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type RevokeUser struct {
	RH string
}

type RevokeUserView struct {
	*RevokeUser
}

func (u *RevokeUserView) Call(context view.Context) (interface{}, error) {
	rh := hash.Hashable(u.RH).String()
	logger.Infof("revoke [%s][%s]", u.RH, rh)
	kvsInstance := kvs.GetService(context)
	k := kvs.CreateCompositeKeyOrPanic("revocationList", []string{rh})
	assert.False(kvsInstance.Exists(k), "Identity already in revoked state")
	assert.NoError(kvsInstance.Put(k, rh), "failed to put revocation handle")
	return nil, nil
}

type RevokeUserViewFactory struct{}

func (u *RevokeUserViewFactory) NewView(in []byte) (view.View, error) {
	f := &RevokeUserView{RevokeUser: &RevokeUser{}}
	err := json.Unmarshal(in, f.RevokeUser)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type RevocationHandle struct {
	RH string
}

type GetRevocationHandle struct {
	TMSID  token.TMSID
	Wallet string
}

type GetRevocationHandleView struct {
	*GetRevocationHandle
}

func (r *GetRevocationHandle) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	w := tms.WalletManager().OwnerWallet(r.Wallet)
	assert.NotNil(w, "wallet not found [%s]", r.Wallet)
	id, err := w.GetRecipientIdentity()
	assert.NoError(err, "error getting recipient id")
	rh, err := tms.WalletManager().GetRevocationHandle(id)
	logger.Infof("RH for [%s] is [%s]", r.Wallet, hash.Hashable(rh).String())
	return &RevocationHandle{RH: rh}, err
}

type GetRevocationHandleViewFactory struct{}

func (p *GetRevocationHandleViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetRevocationHandleView{GetRevocationHandle: &GetRevocationHandle{}}
	err := json.Unmarshal(in, f.GetRevocationHandle)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
