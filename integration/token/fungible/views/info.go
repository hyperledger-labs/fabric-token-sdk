/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type GetEnrollmentID struct {
	Wallet string
	TMSID  *token.TMSID
}

// GetEnrollmentIDView is a view that returns the enrollment ID of a wallet.
type GetEnrollmentIDView struct {
	*GetEnrollmentID
}

func (r *GetEnrollmentIDView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, serviceOpts(r.TMSID)...)
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	w := tms.WalletManager().OwnerWallet(r.Wallet)
	assert.NotNil(w, "wallet not found [%s]", r.Wallet)
	return w.EnrollmentID(), nil
}

type GetEnrollmentIDViewFactory struct{}

func (p *GetEnrollmentIDViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetEnrollmentIDView{GetEnrollmentID: &GetEnrollmentID{}}
	err := json.Unmarshal(in, f.GetEnrollmentID)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type CheckPublicParamsMatch struct {
	TMSID *token.TMSID
}

type CheckPublicParamsMatchView struct {
	*CheckPublicParamsMatch
}

func (p *CheckPublicParamsMatchView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, serviceOpts(p.TMSID)...)
	assert.NotNil(tms, "failed to get TMS")

	assert.NoError(tms.PublicParametersManager().Validate(), "failed to validate local public parameters")
	ppRaw := GetTMSPublicParams(tms)

	fetchedPPRaw, err := tms.PublicParametersManager().Fetch()

	assert.NoError(err, "failed to fetch public params")
	ppm, err := token.NewPublicParametersManagerFromPublicParams(fetchedPPRaw)

	assert.NoError(err, "failed to instantiate public params manager from fetch params")
	assert.NoError(ppm.Validate(), "failed to validate remote public parameters")

	assert.Equal(fetchedPPRaw, ppRaw, "public params do not match [%s]!=[%s]", hash.Hashable(fetchedPPRaw), hash.Hashable(ppRaw))

	return nil, nil
}

type CheckPublicParamsMatchViewFactory struct{}

func (p *CheckPublicParamsMatchViewFactory) NewView(in []byte) (view.View, error) {
	f := &CheckPublicParamsMatchView{CheckPublicParamsMatch: &CheckPublicParamsMatch{}}
	err := json.Unmarshal(in, f.CheckPublicParamsMatch)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type WhoDeletedTokenResult struct {
	Who     []string
	Deleted []bool
}

type WhoDeletedToken struct {
	TMSID    token.TMSID
	TokenIDs []*token2.ID
}

type WhoDeletedTokenView struct {
	*WhoDeletedToken
}

func (w *WhoDeletedTokenView) Call(context view.Context) (interface{}, error) {
	net := network.GetInstance(context, w.TMSID.Network, w.TMSID.Channel)
	assert.NotNil(net, "cannot find network [%s:%s]", w.TMSID.Network, w.TMSID.Channel)
	vault, err := net.Vault(w.TMSID.Namespace)
	assert.NoError(err, "failed to get vault for [%s:%s:%s]", w.TMSID.Network, w.TMSID.Channel, w.TMSID.Namespace)

	who, deleted, err := vault.TokenVault().QueryEngine().WhoDeletedTokens(w.TokenIDs...)
	assert.NoError(err, "failed to lookup who deleted tokens")

	return &WhoDeletedTokenResult{
		Who:     who,
		Deleted: deleted,
	}, nil
}

type WhoDeletedTokenViewFactory struct{}

func (p *WhoDeletedTokenViewFactory) NewView(in []byte) (view.View, error) {
	f := &WhoDeletedTokenView{WhoDeletedToken: &WhoDeletedToken{}}
	err := json.Unmarshal(in, f.WhoDeletedToken)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type GetPublicParams struct {
	TMSID token.TMSID
}

type GetPublicParamsView struct {
	*GetPublicParams
}

func (p *GetPublicParamsView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(p.TMSID))
	assert.NotNil(tms, "failed to get TMS")
	return GetTMSPublicParams(tms), nil
}

type GetPublicParamsViewFactory struct{}

func (p *GetPublicParamsViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetPublicParamsView{GetPublicParams: &GetPublicParams{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

func GetTMSPublicParams(tms *token.ManagementService) []byte {
	ppBytes, err := tms.PublicParametersManager().PublicParameters().Serialize()
	assert.NoError(err, "failed to marshal public params")
	return ppBytes
}

type GetIssuerWalletIdentity struct {
	ID string
}

type GetIssuerWalletIdentityView struct {
	*GetIssuerWalletIdentity
}

type GetIssuerWalletIdentityViewFactory struct{}

func (g *GetIssuerWalletIdentity) Call(context view.Context) (interface{}, error) {
	defaultIssuerWallet := token.GetManagementService(context).WalletManager().IssuerWallet(g.ID)
	assert.NotNil(defaultIssuerWallet, "no default issuer wallet")
	id, err := defaultIssuerWallet.GetIssuerIdentity("")
	assert.NoError(err, "failed getting issuer Identity ")
	return id, err
}

func (p *GetIssuerWalletIdentityViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetIssuerWalletIdentityView{GetIssuerWalletIdentity: &GetIssuerWalletIdentity{}}
	err := json.Unmarshal(in, f.GetIssuerWalletIdentity)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
