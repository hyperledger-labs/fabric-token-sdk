/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

func CompileServiceOptions(opts ...token.ServiceOption) (*token.ServiceOptions, error) {
	txOptions := &token.ServiceOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

// WithRecipientData is used to add a RecipientData to the service options
func WithRecipientData(recipientData *RecipientData) token.ServiceOption {
	return func(options *token.ServiceOptions) error {
		if options.Params == nil {
			options.Params = map[string]interface{}{}
		}
		options.Params["RecipientData"] = recipientData
		return nil
	}
}

func getRecipientData(opts *token.ServiceOptions) *RecipientData {
	rdBoxed, ok := opts.Params["RecipientData"]
	if !ok {
		return nil
	}
	return rdBoxed.(*RecipientData)
}

type RecipientData = token.RecipientData

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
}

func (r *RecipientRequest) Bytes() ([]byte, error) {
	return Marshal(r)
}

func (r *RecipientRequest) FromBytes(raw []byte) error {
	return Unmarshal(raw, r)
}

type RequestRecipientIdentityView struct {
	TMSID              token.TMSID
	Other              view.Identity
	OtherRecipientData *RecipientData
}

// RequestRecipientIdentity executes the RequestRecipientIdentityView.
// The sender contacts the recipient's FSC node identified via the passed view identity.
// The sender gets back the identity the recipient wants to use to assign ownership of tokens.
func RequestRecipientIdentity(context view.Context, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}
	pseudonymBoxed, err := context.RunView(&RequestRecipientIdentityView{
		TMSID:              options.TMSID(),
		Other:              recipient,
		OtherRecipientData: getRecipientData(options),
	})
	if err != nil {
		return nil, err
	}
	return pseudonymBoxed.(view.Identity), nil
}

func (f *RequestRecipientIdentityView) Call(context view.Context) (interface{}, error) {
	w := token.GetManagementService(context, token.WithTMSID(f.TMSID)).WalletManager().OwnerWalletByIdentity(f.Other)

	if isSameNode := w != nil; !isSameNode {
		return f.callWithRecipientData(context)
	}
	if isRemoteRecipient := f.OtherRecipientData != nil; isRemoteRecipient {
		return f.OtherRecipientData.Identity, nil
	}
	return w.GetRecipientIdentity()
}

func (f *RequestRecipientIdentityView) callWithRecipientData(context view.Context) (interface{}, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("request recipient [%s] is not registered", f.Other)
	}
	session, err := context.GetSession(context.Initiator(), f.Other)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session with [%s]", f.Other)
	}

	// Ask for identity
	rr := &RecipientRequest{
		TMSID:         f.TMSID,
		WalletID:      f.Other,
		RecipientData: f.OtherRecipientData,
	}
	rrRaw, err := rr.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling recipient request")
	}
	err = session.Send(rrRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send recipient request")
	}

	// Wait to receive a view identity
	ch := session.Receive()
	var payload []byte

	timeout := time.NewTimer(time.Minute)
	defer timeout.Stop()

	select {
	case msg := <-ch:
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}
		payload = msg.Payload
	case <-timeout.C:
		return nil, errors.New("timeout reached")
	}

	recipientData, err := RecipientDataFromBytes(payload)
	if err != nil {
		logger.Errorf("failed to unmarshal recipient data: [%s][%s]", payload, err)
		return nil, errors.Wrapf(err, "failed to unmarshal recipient data")
	}
	wm := token.GetManagementService(context, token.WithTMSID(f.TMSID)).WalletManager()
	if err := wm.RegisterRecipientIdentity(recipientData); err != nil {
		logger.Errorf("failed to register recipient identity: [%s]", err)
		return nil, errors.Wrapf(err, "failed to register recipient identity")
	}

	// Update the Endpoint Resolver
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("update endpoint resolver for [%s], bind to [%s]", recipientData.Identity, f.Other)
	}
	if err := view2.GetEndpointService(context).Bind(f.Other, recipientData.Identity); err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed binding [%s] to [%s]", recipientData.Identity, f.Other)
		}
		return nil, errors.Wrapf(err, "failed binding [%s] to [%s]", recipientData.Identity, f.Other)
	}

	return recipientData.Identity, nil
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
	session, payload, err := session2.ReadFirstMessage(context)
	if err != nil {
		logger.Errorf("failed to read first message: [%s]", err)
		return nil, errors.Wrapf(err, "failed to read first message")
	}

	recipientRequest := &RecipientRequest{}
	if err := recipientRequest.FromBytes(payload); err != nil {
		logger.Errorf("failed to unmarshal recipient request: [%s][%s]", payload, err)
		return nil, errors.Wrapf(err, "failed to umarshal recipient request")
	}

	wallet := s.Wallet
	if len(wallet) == 0 && len(recipientRequest.WalletID) != 0 {
		wallet = string(recipientRequest.WalletID)
	}
	logger.Debugf("Respond request recipient identity using wallet [%s]", wallet)
	w := GetWallet(
		context,
		wallet,
		token.WithTMSID(recipientRequest.TMSID),
	)
	if w == nil {
		logger.Errorf("failed to get wallet [%s]", wallet)
		return nil, errors.Errorf("wallet [%s:%s] not found", wallet, recipientRequest.TMSID)
	}

	var recipientData *RecipientData
	var recipientIdentity view.Identity
	// if the initiator send a recipient data, check that the identity has been already registered locally.
	if recipientRequest.RecipientData != nil {
		// check it exists and return it back
		recipientData = recipientRequest.RecipientData
		recipientIdentity = recipientData.Identity
		if !w.Contains(recipientIdentity) {
			return nil, errors.Errorf("cannot find identity [%s] in wallet [%s:%s]", recipientIdentity, wallet, recipientRequest.TMSID)
		}
		// TODO: check the other values too
	} else {
		// otherwise generate one fresh
		recipientIdentity, err = w.GetRecipientIdentity()
		if err != nil {
			logger.Errorf("failed to get recipient identity: [%s]", err)
			return nil, errors.Wrapf(err, "failed to get recipient identity")
		}
		auditInfo, err := w.GetAuditInfo(recipientIdentity)
		if err != nil {
			logger.Errorf("failed to get audit info: [%s]", err)
			return nil, errors.Wrapf(err, "failed to get audit info")
		}
		metadata, err := w.GetTokenMetadata(recipientIdentity)
		if err != nil {
			logger.Errorf("failed to get token metadata: [%s]", err)
			return nil, errors.Wrapf(err, "failed to get token metadata")
		}
		recipientData = &RecipientData{
			Identity:      recipientIdentity,
			AuditInfo:     auditInfo,
			TokenMetadata: metadata,
		}
	}
	recipientDataRaw, err := RecipientDataBytes(recipientData)
	if err != nil {
		logger.Errorf("failed to marshal recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed marshalling recipient data")
	}

	// Step 3: send the public key back to the invoker
	err = session.Send(recipientDataRaw)
	if err != nil {
		logger.Errorf("failed to send recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	// Update the Endpoint Resolver
	resolver := view2.GetEndpointService(context)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("bind me [%s] to [%s]", context.Me(), recipientData)
	}
	err = resolver.Bind(context.Me(), recipientIdentity)
	if err != nil {
		logger.Errorf("failed binding [%s] to [%s]", context.Me(), recipientData)
		return nil, errors.Wrapf(err, "failed to bind me to recipient identity")
	}

	return recipientIdentity, nil
}

type ExchangeRecipientIdentitiesView struct {
	TMSID  token.TMSID
	Wallet string
	Other  view.Identity
}

func (f *ExchangeRecipientIdentitiesView) Call(context view.Context) (interface{}, error) {
	ts := token.GetManagementService(context, token.WithTMSID(f.TMSID))

	if w := ts.WalletManager().OwnerWalletByIdentity(f.Other); w != nil {
		other, err := w.GetRecipientIdentity()
		if err != nil {
			return nil, err
		}

		me, err := ts.WalletManager().OwnerWallet(f.Wallet).GetRecipientIdentity()
		if err != nil {
			return nil, err
		}

		return []view.Identity{me, other}, nil
	} else {
		session, err := context.GetSession(context.Initiator(), f.Other)
		if err != nil {
			return nil, err
		}

		w := ts.WalletManager().OwnerWallet(f.Wallet)
		me, err := w.GetRecipientIdentity()
		if err != nil {
			return nil, err
		}
		auditInfo, err := w.GetAuditInfo(me)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting recipient identity audit info, wallet [%s]", w.ID())
		}
		metadata, err := w.GetTokenMetadata(me)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting recipient identity metadata, wallet [%s]", w.ID())
		}
		// Send request
		request := &ExchangeRecipientRequest{
			TMSID:    f.TMSID,
			WalletID: f.Other,
			RecipientData: &RecipientData{
				Identity:      me,
				AuditInfo:     auditInfo,
				TokenMetadata: metadata,
			},
		}
		requestRaw, err := request.Bytes()
		if err != nil {
			return nil, err
		}
		if err := session.Send(requestRaw); err != nil {
			return nil, err
		}

		// Wait to receive a view identity
		payload, err := session2.ReadMessageWithTimeout(session, 30*time.Second)
		if err != nil {
			return nil, err
		}

		recipientData, err := RecipientDataFromBytes(payload)
		if err != nil {
			return nil, err
		}
		if err := ts.WalletManager().RegisterRecipientIdentity(recipientData); err != nil {
			return nil, err
		}

		// Update the Endpoint Resolver
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("bind [%s] to other [%s]", recipientData.Identity, f.Other)
		}
		resolver := view2.GetEndpointService(context)
		err = resolver.Bind(f.Other, recipientData.Identity)
		if err != nil {
			return nil, err
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("bind me [%s] to [%s]", me, context.Me())
		}
		err = resolver.Bind(context.Me(), me)
		if err != nil {
			return nil, err
		}

		return []view.Identity{me, recipientData.Identity}, nil
	}
}

// ExchangeRecipientIdentities executes the ExchangeRecipientIdentitiesView using by passed wallet id to
// derive the recipient identity to send to the passed recipient.
// The function returns, the recipient identity of the sender, the recipient identity of the recipient
func ExchangeRecipientIdentities(context view.Context, walletID string, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, view.Identity, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, nil, err
	}
	ids, err := context.RunView(&ExchangeRecipientIdentitiesView{
		TMSID:  options.TMSID(),
		Wallet: walletID,
		Other:  recipient,
	})
	if err != nil {
		return nil, nil, err
	}

	return ids.([]view.Identity)[0], ids.([]view.Identity)[1], nil
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
	session, requestRaw, err := session2.ReadFirstMessage(context)
	if err != nil {
		return nil, err
	}

	// other
	request := &ExchangeRecipientRequest{}
	if err := request.FromBytes(requestRaw); err != nil {
		return nil, err
	}

	ts := token.GetManagementService(context, token.WithTMSID(request.TMSID))
	other := request.RecipientData.Identity
	if err := ts.WalletManager().RegisterRecipientIdentity(&RecipientData{
		Identity: other, AuditInfo: request.RecipientData.AuditInfo, TokenMetadata: request.RecipientData.TokenMetadata,
	}); err != nil {
		return nil, err
	}

	// me
	wallet := s.Wallet
	if len(wallet) == 0 && len(request.WalletID) != 0 {
		wallet = string(request.WalletID)
	}
	w := ts.WalletManager().OwnerWallet(wallet)
	me, err := w.GetRecipientIdentity()
	if err != nil {
		return nil, err
	}
	auditInfo, err := w.GetAuditInfo(me)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient identity audit info, wallet [%s]", w.ID())
	}
	metadata, err := w.GetTokenMetadata(me)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient identity metadata, wallet [%s]", w.ID())
	}

	recipientData := &RecipientData{
		Identity:      me,
		AuditInfo:     auditInfo,
		TokenMetadata: metadata,
	}
	recipientDataRaw, err := RecipientDataBytes(recipientData)
	if err != nil {
		return nil, err
	}

	if err := session.Send(recipientDataRaw); err != nil {
		return nil, err
	}

	// Update the Endpoint Resolver
	resolver := view2.GetEndpointService(context)
	err = resolver.Bind(context.Me(), me)
	if err != nil {
		return nil, err
	}
	err = resolver.Bind(session.Info().Caller, other)
	if err != nil {
		return nil, err
	}

	return []view.Identity{me, other}, nil
}
