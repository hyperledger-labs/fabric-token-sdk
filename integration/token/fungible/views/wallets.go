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
)

type RegisterIssuerWallet struct {
	TMSID token.TMSID
	ID    string
	Path  string
}

// RegisterIssuerWalletView is a view that register an issuer wallet
type RegisterIssuerWalletView struct {
	*RegisterIssuerWallet
}

func (r *RegisterIssuerWalletView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	err := tms.WalletManager().RegisterIssuerWallet(r.ID, r.Path)
	assert.NoError(err, "failed to register issuer wallet [%s:%s]", r.ID, r.TMSID)
	return nil, nil
}

type RegisterIssuerWalletViewFactory struct{}

func (p *RegisterIssuerWalletViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterIssuerWalletView{RegisterIssuerWallet: &RegisterIssuerWallet{}}
	err := json.Unmarshal(in, f.RegisterIssuerWallet)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type RegisterOwnerWallet struct {
	TMSID token.TMSID
	ID    string
	Path  string
}

// RegisterOwnerWalletView is a view that registers an owner wallet
type RegisterOwnerWalletView struct {
	*RegisterOwnerWallet
}

func (r *RegisterOwnerWalletView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	err := tms.WalletManager().RegisterOwnerWallet(r.ID, r.Path)
	assert.NoError(err, "failed to register owner wallet [%s:%s]", r.ID, r.TMSID)
	return nil, nil
}

type RegisterOwnerWalletViewFactory struct{}

func (p *RegisterOwnerWalletViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterOwnerWalletView{RegisterOwnerWallet: &RegisterOwnerWallet{}}
	err := json.Unmarshal(in, f.RegisterOwnerWallet)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type RegisterRecipientData struct {
	TMSID         token.TMSID
	WalletID      string
	RecipientData token.RecipientData
}

// RegisterRecipientDataView is a view that registers recipient data
type RegisterRecipientDataView struct {
	*RegisterRecipientData
}

func (r *RegisterRecipientDataView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	err := tms.WalletManager().OwnerWallet(r.WalletID).RegisterRecipient(&r.RecipientData)
	assert.NoError(err, "failed to register recipient data [%s:%s]", r.WalletID, r.TMSID)
	return nil, nil
}

type RegisterRecipientDataViewFactory struct{}

func (p *RegisterRecipientDataViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterRecipientDataView{RegisterRecipientData: &RegisterRecipientData{}}
	err := json.Unmarshal(in, f.RegisterRecipientData)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
