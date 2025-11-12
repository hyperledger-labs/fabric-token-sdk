/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	view3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/view"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

type RecipientData = token.RecipientData

type MultisigRecipientData struct {
	RecipientData *token.RecipientData
	Nodes         []view.Identity
	Recipients    []token.Identity
}

type ExchangeRecipientRequest struct {
	TMSID         token.TMSID
	WalletID      []byte
	RecipientData *RecipientData
}

func (r *ExchangeRecipientRequest) Bytes() ([]byte, error) {
	return Marshal(r)
}

func (r *ExchangeRecipientRequest) FromBytes(raw []byte) error {
	return Unmarshal(raw, r)
}

type RecipientRequest struct {
	TMSID         token.TMSID
	WalletID      []byte
	RecipientData *RecipientData
	MultiSig      bool
}

func (r *RecipientRequest) Bytes() ([]byte, error) {
	return Marshal(r)
}

func (r *RecipientRequest) FromBytes(raw []byte) error {
	return Unmarshal(raw, r)
}

type Recipient struct {
	Identity      view.Identity
	WalletID      string
	RecipientData *RecipientData
}

type Recipients []Recipient

func (r Recipients) Identities() []view.Identity {
	ids := make([]view.Identity, len(r))
	for i, recipient := range r {
		ids[i] = recipient.Identity
	}
	return ids
}

type RequestRecipientIdentityView struct {
	TMSID      token.TMSID
	Recipients Recipients
}

// RequestRecipientIdentity executes the RequestRecipientIdentityView.
// The sender contacts the recipient's FSC node identified via the passed view identity.
// The sender gets back the identity the recipient wants to use to assign ownership of tokens.
func RequestRecipientIdentity(context view.Context, recipient view.Identity, opts ...token.ServiceOption) (token.Identity, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}
	pseudonymBoxed, err := view3.RunViewWithTimeout(
		context,
		&RequestRecipientIdentityView{
			TMSID: options.TMSID(),
			Recipients: []Recipient{
				{
					Identity:      recipient,
					RecipientData: getRecipientData(options),
					WalletID:      getRecipientWalletID(options),
				},
			},
		},
		options.Duration,
	)
	if err != nil {
		return nil, err
	}
	return pseudonymBoxed.(view.Identity), nil
}

// RequestMultisigIdentity collects the recipient identities from all the passed identities.
// It merges them into a single multisig identity and distributes it to all the participants.
func RequestMultisigIdentity(context view.Context, ids []view.Identity, opts ...token.ServiceOption) (token.Identity, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling service options")
	}
	recipients := make([]Recipient, len(ids))
	for i, id := range ids {
		recipients[i] = Recipient{
			Identity:      id,
			RecipientData: getRecipientData(options),
		}
	}
	pseudonymBoxed, err := view3.RunViewWithTimeout(
		context,
		&RequestRecipientIdentityView{
			TMSID:      options.TMSID(),
			Recipients: recipients,
		},
		options.Duration,
	)
	if err != nil {
		return nil, err
	}
	return pseudonymBoxed.(view.Identity), nil
}

func (f *RequestRecipientIdentityView) Call(context view.Context) (interface{}, error) {
	results := make([]token.Identity, len(f.Recipients))
	local := make([]bool, len(f.Recipients))
	var err error
	tms, err := token.GetManagementService(context, token.WithTMSID(f.TMSID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting token management service [%s]", f.TMSID)
	}
	multiSig := len(f.Recipients) > 1
	for i, recipient := range f.Recipients {
		local[i] = true
		w := tms.WalletManager().OwnerWallet(context.Context(), recipient.Identity)

		if isSameNode := w != nil; !isSameNode {
			results[i], err = f.callWithRecipientData(context, &recipient, multiSig)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get recipient identity")
			}
			local[i] = false
			continue
		}

		if isRemoteRecipient := recipient.RecipientData != nil; isRemoteRecipient {
			results[i] = recipient.RecipientData.Identity
			continue
		}
		if w == nil {
			return nil, errors.Errorf("wallet [%s] not found", string(recipient.Identity))
		}
		results[i], err = w.GetRecipientIdentity(context.Context())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get recipient identity")
		}
	}
	if !multiSig {
		return results[0], nil
	}

	// aggregate the results as multisig identity, then distribute the aggregate results to all the participants
	multisigIdentity, err := f.aggregateAndDistribute(context, tms, results, local)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to aggregate recipient identities")
	}
	return multisigIdentity, nil
}

func (f *RequestRecipientIdentityView) callWithRecipientData(context view.Context, recipient *Recipient, multiSig bool) (token.Identity, error) {
	logger.DebugfContext(context.Context(), "request recipient [%s] is not registered", recipient.Identity)
	session, err := session2.NewFromInitiator(context, recipient.Identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session with [%s]", recipient.Identity)
	}

	// Ask for identity
	wID := []byte(recipient.WalletID)
	if len(wID) == 0 {
		wID = recipient.Identity
	}
	recipientRequest := &RecipientRequest{
		TMSID:         f.TMSID,
		WalletID:      wID,
		RecipientData: recipient.RecipientData,
		MultiSig:      multiSig,
	}
	logger.DebugfContext(context.Context(), "Send identity request to %s", wID)
	err = session.Send(recipientRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send recipient request")
	}

	logger.DebugfContext(context.Context(), "Receive identity response")
	recipientData := &RecipientData{}
	err = session.ReceiveWithTimeout(recipientData, 10*time.Second)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal recipient data")
	}
	tms, err := token.GetManagementService(context, token.WithTMSID(f.TMSID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token management service")
	}
	wm := tms.WalletManager()
	logger.DebugfContext(context.Context(), "Register recipient identity")
	if err := wm.RegisterRecipientIdentity(context.Context(), recipientData); err != nil {
		return nil, errors.Wrapf(err, "failed to register recipient identity")
	}

	// Update the Endpoint Resolver
	logger.DebugfContext(context.Context(), "update endpoint resolver for [%s], bind to [%s]", recipientData.Identity, recipient.Identity)

	if err := endpoint.GetService(context).Bind(context.Context(), recipient.Identity, recipientData.Identity); err != nil {
		logger.ErrorfContext(context.Context(), "failed binding [%s] to [%s]: %s", recipientData.Identity, recipient.Identity, err)
		return nil, errors.Wrapf(err, "failed binding [%s] to [%s]", recipientData.Identity, recipient.Identity)
	}
	return recipientData.Identity, nil
}

func (f *RequestRecipientIdentityView) aggregateAndDistribute(context view.Context, tms *token.ManagementService, recipients []token.Identity, local []bool) (token.Identity, error) {
	// prepare identity
	multisigIdentity, err := multisig.WrapIdentities(recipients...)
	if err != nil {
		return nil, errors.Wrap(err, "failed wrapping identities")
	}

	// prepare audit info
	auditInfoForRecipients, err := tms.SigService().GetAuditInfo(context.Context(), recipients...)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting token audit info")
	}
	auditInfo, err := multisig.WrapAuditInfo(auditInfoForRecipients)
	if err != nil {
		return nil, errors.Wrap(err, "failed wrapping audit info")
	}

	// register audit info for the recipient
	recipientData := &token.RecipientData{
		Identity:  multisigIdentity,
		AuditInfo: auditInfo,
	}
	err = tms.WalletManager().RegisterRecipientIdentity(context.Context(), recipientData)
	if err != nil {
		return nil, errors.Wrapf(err, "failed registering recipient identity [%s]", multisigIdentity)
	}

	// distribute recipient identity and its audit info to all the participants
	mrd := &MultisigRecipientData{
		RecipientData: recipientData,
		Nodes:         f.Recipients.Identities(),
		Recipients:    recipients,
	}
	for i, recipient := range f.Recipients {
		if local[i] {
			continue
		}
		session, err := session2.NewJSON(context, context.Initiator(), recipient.Identity)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get session with [%s]", recipient.Identity)
		}
		err = session.Send(mrd)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to send recipient request")
		}
	}
	return multisigIdentity, nil
}

type RespondRequestRecipientIdentityView struct {
	Wallet string
}

// RespondRequestRecipientIdentity executes the RespondRequestRecipientIdentityView.
// The recipient sends back the identity to receive ownership of tokens.
// The identity is taken from the default wallet.
// If the wallet is not found, an error is returned.
func RespondRequestRecipientIdentity(context view.Context) (view.Identity, error) {
	return RespondRequestRecipientIdentityUsingWallet(context, "")
}

// RespondRequestRecipientIdentityUsingWallet executes the RespondRequestRecipientIdentityView.
// The recipient sends back the identity to receive ownership of tokens.
// The identity is taken from the passed wallet.
// If the wallet is not found, an error is returned.
// If the wallet is the empty string, the identity is taken from the default wallet.
func RespondRequestRecipientIdentityUsingWallet(context view.Context, wallet string) (view.Identity, error) {
	id, err := context.RunView(&RespondRequestRecipientIdentityView{Wallet: wallet})
	if err != nil {
		return nil, err
	}
	return id.(view.Identity), nil
}

func (s *RespondRequestRecipientIdentityView) Call(context view.Context) (interface{}, error) {
	session := session2.JSON(context)
	recipientRequest := &RecipientRequest{}
	if err := session.Receive(recipientRequest); err != nil {
		return nil, errors.Wrapf(err, "failed to receive recipient request")
	}

	wallet := s.Wallet
	if len(wallet) == 0 && len(recipientRequest.WalletID) != 0 {
		wallet = string(recipientRequest.WalletID)
	}
	logger.DebugfContext(context.Context(), "Respond request recipient identity using wallet [%s]", wallet)
	tms, err := token.GetManagementService(context, token.WithTMSID(recipientRequest.TMSID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting token management service [%s]", recipientRequest.TMSID)
	}
	w := tms.WalletManager().OwnerWallet(context.Context(), wallet)
	if w == nil {
		return nil, errors.Errorf("wallet [%s:%s] not found", wallet, recipientRequest.TMSID)
	}

	var recipientData *RecipientData
	var recipientIdentity view.Identity
	// if the initiator send a recipient data, check that the identity has been already registered locally.
	if recipientRequest.RecipientData != nil {
		// check it exists and return it back
		recipientData = recipientRequest.RecipientData
		recipientIdentity = recipientData.Identity
		if !w.Contains(context.Context(), recipientIdentity) {
			return nil, errors.Errorf("cannot find identity [%s] in wallet [%s:%s]", recipientIdentity, wallet, recipientRequest.TMSID)
		}
		// TODO: check the other values too
	} else {
		logger.DebugfContext(context.Context(), "generate_identity")
		// otherwise generate one fresh
		var err error
		recipientData, err = w.GetRecipientData(context.Context())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get recipient identity")
		}
		recipientIdentity = recipientData.Identity
	}

	// Step 3: send the public key back to the invoker
	logger.DebugfContext(context.Context(), "Send recipient identity response to %s", session.Info().Caller)
	if err := session.Send(recipientData); err != nil {
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	// Update the Endpoint Resolver
	resolver := endpoint.GetService(context)
	logger.DebugfContext(context.Context(), "bind me [%s] to [%s]", context.Me(), recipientData)

	err = resolver.Bind(context.Context(), context.Me(), recipientIdentity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to bind me to recipient identity")
	}

	if err := s.handleMultisig(context, session.Session(), tms, recipientRequest, recipientIdentity); err != nil {
		return nil, errors.Wrapf(err, "failed to handle multisig")
	}

	return recipientIdentity, nil
}

func (s *RespondRequestRecipientIdentityView) handleMultisig(
	context view.Context,
	session view.Session,
	tms *token.ManagementService,
	recipientRequest *RecipientRequest,
	recipientIdentity token.Identity,
) error {
	if !recipientRequest.MultiSig {
		logger.DebugfContext(context.Context(), "Skip multisig")
		return nil
	}

	jsonSession := session2.NewFromSession(context, session)

	logger.DebugfContext(context.Context(), "Receive multisig")
	multisigRecipientData := &MultisigRecipientData{}
	err := jsonSession.Receive(multisigRecipientData)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal multisig recipient data")
	}
	logger.DebugfContext(context.Context(), "Received multisig")

	// unmarshal the envelope

	// register the multisig recipient identity
	wm := tms.WalletManager()
	err = wm.RegisterRecipientIdentity(context.Context(), multisigRecipientData.RecipientData)
	if err != nil {
		return errors.Wrapf(err, "failed to register recipient identity")
	}
	sigService := tms.SigService()
	signer, err := sigService.GetSigner(context.Context(), recipientIdentity)
	if err != nil {
		return err
	}
	logger.DebugfContext(context.Context(), "registering signer for reclaim...")
	if err := sigService.RegisterSigner(
		context.Context(),
		multisigRecipientData.RecipientData.Identity,
		signer,
		nil,
	); err != nil {
		return err
	}

	// register the audit info for each party too
	multisigIdentities, ok, err := multisig.Unwrap(multisigRecipientData.RecipientData.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed to unwrap multisig identity")
	}
	if !ok {
		return errors.Errorf("expected multisig identity")
	}
	ok, auditInfos, err := multisig.UnwrapAuditInfo(multisigRecipientData.RecipientData.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to unwrap multisig audit info")
	}
	if !ok {
		return errors.Errorf("expected multisig audit info")
	}
	for i, identity := range multisigIdentities {
		if identity.Equal(recipientIdentity) {
			continue
		}
		err = wm.RegisterRecipientIdentity(context.Context(), &RecipientData{
			Identity:               identity,
			AuditInfo:              auditInfos[i],
			TokenMetadata:          multisigRecipientData.RecipientData.TokenMetadata,
			TokenMetadataAuditInfo: multisigRecipientData.RecipientData.TokenMetadataAuditInfo,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to register recipient identity")
		}
	}

	// Update the Endpoint Resolver
	resolver := endpoint.GetService(context)
	for i, node := range multisigRecipientData.Nodes {
		err = resolver.Bind(context.Context(), node, multisigRecipientData.Recipients[i])
		if err != nil {
			return errors.Wrapf(err, "failed to bind me to recipient identity")
		}
	}
	return nil
}

type ExchangeRecipientIdentitiesView struct {
	TMSID  token.TMSID
	Wallet string
	Other  view.Identity
}

// ExchangeRecipientIdentities executes the ExchangeRecipientIdentitiesView using by passed wallet id to
// derive the recipient identity to send to the passed recipient.
// The function returns, the recipient identity of the sender, the recipient identity of the recipient
func ExchangeRecipientIdentities(context view.Context, walletID string, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, view.Identity, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed compiling service options")
	}
	ids, err := context.RunView(&ExchangeRecipientIdentitiesView{
		TMSID:  options.TMSID(),
		Wallet: walletID,
		Other:  recipient,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed running view")
	}

	return ids.([]view.Identity)[0], ids.([]view.Identity)[1], nil
}

func (f *ExchangeRecipientIdentitiesView) Call(context view.Context) (interface{}, error) {
	ts, err := token.GetManagementService(context, token.WithTMSID(f.TMSID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token management service")
	}

	if w := ts.WalletManager().OwnerWallet(context.Context(), f.Other); w != nil {
		other, err := w.GetRecipientIdentity(context.Context())
		if err != nil {
			return nil, err
		}

		me, err := ts.WalletManager().OwnerWallet(context.Context(), f.Wallet).GetRecipientIdentity(context.Context())
		if err != nil {
			return nil, err
		}

		return []view.Identity{me, other}, nil
	} else {
		session, err := session2.NewFromInitiator(context, f.Other)
		if err != nil {
			return nil, err
		}

		w := ts.WalletManager().OwnerWallet(context.Context(), f.Wallet)
		if w == nil {
			return nil, errors.Errorf("wallet [%s:%s] not found", f.Wallet, f.TMSID)
		}
		localRecipientData, err := w.GetRecipientData(context.Context())
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting recipient data, wallet [%s]", w.ID())
		}
		// Send request
		request := &ExchangeRecipientRequest{
			TMSID:         f.TMSID,
			WalletID:      f.Other,
			RecipientData: localRecipientData,
		}
		if err := session.Send(request); err != nil {
			return nil, err
		}

		// Wait to receive a *token.RecipientData
		remoteRecipientData := &token.RecipientData{}
		err = session.Receive(remoteRecipientData)
		if err != nil {
			return nil, err
		}

		if err := ts.WalletManager().RegisterRecipientIdentity(context.Context(), remoteRecipientData); err != nil {
			return nil, err
		}

		// Update the Endpoint Resolver
		logger.DebugfContext(context.Context(), "bind [%s] to other [%s]", remoteRecipientData.Identity, f.Other)
		resolver := endpoint.GetService(context)
		err = resolver.Bind(context.Context(), f.Other, remoteRecipientData.Identity)
		if err != nil {
			return nil, err
		}

		logger.DebugfContext(context.Context(), "bind me [%s] to [%s]", localRecipientData.Identity, context.Me())
		err = resolver.Bind(context.Context(), context.Me(), localRecipientData.Identity)
		if err != nil {
			return nil, err
		}

		return []view.Identity{localRecipientData.Identity, remoteRecipientData.Identity}, nil
	}
}

type RespondExchangeRecipientIdentitiesView struct {
	Wallet string
}

// RespondExchangeRecipientIdentities executes the RespondExchangeRecipientIdentitiesView.
// The recipient sends back the identity to receive ownership of tokens.
// The identity is taken from the default wallet
func RespondExchangeRecipientIdentities(context view.Context) (view.Identity, view.Identity, error) {
	ids, err := context.RunView(&RespondExchangeRecipientIdentitiesView{})
	if err != nil {
		return nil, nil, err
	}

	return ids.([]view.Identity)[0], ids.([]view.Identity)[1], nil
}

func (s *RespondExchangeRecipientIdentitiesView) Call(context view.Context) (interface{}, error) {
	session := session2.JSON(context)

	// other
	request := &ExchangeRecipientRequest{}
	if err := session.Receive(request); err != nil {
		return nil, err
	}

	ts, err := token.GetManagementService(context, token.WithTMSID(request.TMSID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token management service")
	}
	other := request.RecipientData.Identity
	if err := ts.WalletManager().RegisterRecipientIdentity(context.Context(), &RecipientData{
		Identity: other, AuditInfo: request.RecipientData.AuditInfo, TokenMetadata: request.RecipientData.TokenMetadata,
	}); err != nil {
		return nil, err
	}

	// me
	wallet := s.Wallet
	if len(wallet) == 0 && len(request.WalletID) != 0 {
		wallet = string(request.WalletID)
	}
	w := ts.WalletManager().OwnerWallet(context.Context(), wallet)
	if w == nil {
		return nil, errors.Errorf("wallet [%s] not found", wallet)
	}

	recipientData, err := w.GetRecipientData(context.Context())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient data, wallet [%s]", w.ID())
	}

	if err := session.Send(recipientData); err != nil {
		return nil, errors.WithMessagef(err, "failed sending recipient data, wallet [%s]", w.ID())
	}

	// Update the Endpoint Resolver
	resolver := endpoint.GetService(context)
	err = resolver.Bind(context.Context(), context.Me(), recipientData.Identity)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed binding recipient data, wallet [%s]", w.ID())
	}
	err = resolver.Bind(context.Context(), session.Info().Caller, other)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed binding recipient data, wallet [%s]", w.ID())
	}

	return []token.Identity{recipientData.Identity, other}, nil
}

func getRecipientData(opts *token.ServiceOptions) *RecipientData {
	rdBoxed, ok := opts.Params["RecipientData"]
	if !ok {
		return nil
	}
	return rdBoxed.(*RecipientData)
}
