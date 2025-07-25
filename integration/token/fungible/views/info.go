/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	OwnerWallet = iota
	IssuerWallet
	AuditorWallet
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
	tms := token.GetManagementService(context, ServiceOpts(r.TMSID)...)
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	w := tms.WalletManager().OwnerWallet(context.Context(), r.Wallet)
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
	tms := token.GetManagementService(context, ServiceOpts(p.TMSID)...)
	assert.NotNil(tms, "failed to get TMS")

	assert.NotNil(tms.PublicParametersManager().PublicParameters(), "failed to validate local public parameters")

	fetchedPPRaw, err := network.GetInstance(context, tms.Network(), tms.Channel()).FetchPublicParameters(tms.Namespace())
	assert.NoError(err, "failed to fetch public params")
	is := core.NewPPManagerFactoryService(fabtoken.NewPPMFactory(), dlog.NewPPMFactory())
	pp, err := is.PublicParametersFromBytes(fetchedPPRaw)
	assert.NoError(err, "failed deserializing public parameters")
	assert.NotNil(pp)
	assert.NoError(pp.Validate())
	fetchedPPRawHash := token.PPHash(utils.Hashable(fetchedPPRaw).Raw())
	assert.Equal(
		fetchedPPRawHash,
		tms.PublicParametersManager().PublicParamsHash(),
		"public params do not match [%s]!=[%s]",
		logging.Base64(fetchedPPRawHash),
		logging.Base64(tms.PublicParametersManager().PublicParamsHash()),
	)

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
	tms := token.GetManagementService(context, token.WithTMSID(w.TMSID))
	assert.NotNil(tms, "failed to get TMS [%s]", w.TMSID)
	who, deleted, err := tms.Vault().NewQueryEngine().WhoDeletedTokens(context.Context(), w.TokenIDs...)
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

type UpdatePublicParams struct {
	TMSID token.TMSID
}

type UpdatePublicParamsView struct {
	*UpdatePublicParams
}

func (p *UpdatePublicParamsView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, ServiceOpts(&p.TMSID)...)
	assert.NotNil(tms, "failed to get TMS")
	assert.NotNil(tms.PublicParametersManager().PublicParameters(), "failed to validate local public parameters")
	fetchedPPRaw, err := network.GetInstance(context, tms.Network(), tms.Channel()).FetchPublicParameters(tms.Namespace())
	assert.NoError(err, "failed to fetch public parameters")
	assert.NoError(token.GetManagementServiceProvider(context).Update(tms.ID(), fetchedPPRaw), "failed to update public parameters")
	return nil, nil
}

type UpdatePublicParamsViewFactory struct{}

func (p *UpdatePublicParamsViewFactory) NewView(in []byte) (view.View, error) {
	f := &UpdatePublicParamsView{UpdatePublicParams: &UpdatePublicParams{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type DoesWalletExist struct {
	TMSID      token.TMSID
	Wallet     string
	WalletType int
}

type DoesWalletExistView struct {
	*DoesWalletExist
}

func (p *DoesWalletExistView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(p.TMSID))
	assert.NotNil(tms, "failed to get TMS")
	switch p.WalletType {
	case OwnerWallet:
		return tms.WalletManager().OwnerWallet(context.Context(), p.Wallet) != nil, nil
	case IssuerWallet:
		return tms.WalletManager().IssuerWallet(context.Context(), p.Wallet) != nil, nil
	case AuditorWallet:
		return tms.WalletManager().AuditorWallet(context.Context(), p.Wallet) != nil, nil
	default:
		return tms.WalletManager().OwnerWallet(context.Context(), p.Wallet) != nil, nil
	}
}

type DoesWalletExistViewFactory struct{}

func (p *DoesWalletExistViewFactory) NewView(in []byte) (view.View, error) {
	f := &DoesWalletExistView{DoesWalletExist: &DoesWalletExist{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type TxStatus struct {
	TMSID token.TMSID
	TxID  string
}

type TxStatusResponse struct {
	ValidationCode    ttx.TxStatus
	ValidationMessage string
}

type TxStatusView struct {
	*TxStatus
}

func (p *TxStatusView) Call(context view.Context) (interface{}, error) {
	owner := ttx.NewOwner(context, token.GetManagementService(context, token.WithTMSID(p.TMSID)))
	vc, message, err := owner.GetStatus(context.Context(), p.TxID)
	assert.NoError(err, "failed to retrieve status of [%s]", p.TxID)
	return &TxStatusResponse{
		ValidationCode:    vc,
		ValidationMessage: message,
	}, nil
}

type TxStatusViewFactory struct{}

func (p *TxStatusViewFactory) NewView(in []byte) (view.View, error) {
	f := &TxStatusView{TxStatus: &TxStatus{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
