/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxcc

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

func compileServiceOptions(opts ...token.ServiceOption) (*token.ServiceOptions, error) {
	txOptions := &token.ServiceOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

type RecipientData struct {
	Identity  view.Identity
	AuditInfo []byte
	Metadata  []byte
}

func (r *RecipientData) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

func (r *RecipientData) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, r)
}

type ExchangeRecipientRequest struct {
	TMSID         token.TMSID
	WalletID      []byte
	RecipientData *RecipientData
}

func (r *ExchangeRecipientRequest) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

func (r *ExchangeRecipientRequest) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, r)
}

type RecipientRequest struct {
	TMSID    token.TMSID
	WalletID []byte
}

func (r *RecipientRequest) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

func (r *RecipientRequest) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, r)
}

type RequestRecipientIdentityView struct {
	TMSID   token.TMSID
	Other   view.Identity
	Timeout time.Duration
}

// RequestRecipientIdentity executes the RequestRecipientIdentityView.
// The sender contacts the recipient's FSC node identified via the passed view identity.
// The sender gets back the identity the recipient wants to use to assign ownership of tokens.
func RequestRecipientIdentity(context view.Context, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, error) {
	o, err := compileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}
	if o.Timeout == 0 {
		o.Timeout = time.Minute * 10
	}
	pseudonymBoxed, err := context.RunView(&RequestRecipientIdentityView{TMSID: o.TMSID(), Other: recipient, Timeout: o.Timeout})
	if err != nil {
		return nil, err
	}
	return pseudonymBoxed.(view.Identity), nil
}

func (f RequestRecipientIdentityView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("request recipient to [%s] for TMS [%s]", f.Other, f.TMSID)

	tms := token.GetManagementService(context, token.WithTMSID(f.TMSID))

	if w := tms.WalletManager().OwnerWalletByIdentity(f.Other); w != nil {
		recipient, err := w.GetRecipientIdentity()
		if err != nil {
			return nil, err
		}
		return recipient, nil
	} else {
		session, err := context.GetSession(context.Initiator(), f.Other)
		if err != nil {
			return nil, err
		}

		// Ask for identity
		rr := &RecipientRequest{
			TMSID:    f.TMSID,
			WalletID: f.Other,
		}
		rrRaw, err := rr.Bytes()
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshalling recipient request")
		}
		err = session.Send(rrRaw)
		if err != nil {
			return nil, err
		}

		// Wait to receive a view identity
		ch := session.Receive()
		var payload []byte
		select {
		case msg := <-ch:
			payload = msg.Payload
		case <-time.After(f.Timeout):
			return nil, errors.New("time out reached")
		}

		recipientData := &RecipientData{}
		if err := recipientData.FromBytes(payload); err != nil {
			return nil, err
		}
		if err := tms.WalletManager().RegisterRecipientIdentity(recipientData.Identity, recipientData.AuditInfo, recipientData.Metadata); err != nil {
			return nil, err
		}

		// Update the Endpoint Resolver
		if err := view2.GetEndpointService(context).Bind(f.Other, recipientData.Identity); err != nil {
			return nil, err
		}

		return recipientData.Identity, nil
	}
}

type RespondRequestRecipientIdentityView struct {
	Wallet string
}

func (s *RespondRequestRecipientIdentityView) Call(context view.Context) (interface{}, error) {
	session, payload, err := session2.ReadFirstMessage(context)
	if err != nil {
		return nil, err
	}

	recipientRequest := &RecipientRequest{}
	if err := recipientRequest.FromBytes(payload); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling recipient request")
	}

	wallet := s.Wallet
	if len(wallet) == 0 && len(recipientRequest.WalletID) != 0 {
		wallet = string(recipientRequest.WalletID)
	}
	w := GetWallet(
		context,
		wallet,
		token.WithTMSID(recipientRequest.TMSID),
	)
	recipientIdentity, err := w.GetRecipientIdentity()
	if err != nil {
		return nil, err
	}
	auditInfo, err := w.GetAuditInfo(recipientIdentity)
	if err != nil {
		return nil, err
	}
	metadata, err := w.GetTokenMetadata(recipientIdentity)
	if err != nil {
		return nil, err
	}
	recipientData := &RecipientData{
		Identity:  recipientIdentity,
		AuditInfo: auditInfo,
		Metadata:  metadata,
	}
	recipientDataRaw, err := recipientData.Bytes()
	if err != nil {
		return nil, err
	}

	// Step 3: send the public key back to the invoker
	err = session.Send(recipientDataRaw)
	if err != nil {
		return nil, err
	}

	// Update the Endpoint Resolver
	resolver := view2.GetEndpointService(context)
	err = resolver.Bind(context.Me(), recipientIdentity)
	if err != nil {
		return nil, err
	}

	return recipientIdentity, nil
}

// RespondRequestRecipientIdentity executes the RespondRequestRecipientIdentityView.
// The recipient sends back the identity to receive ownership of tokens.
// The identity is taken from the wallet
func RespondRequestRecipientIdentity(context view.Context) (view.Identity, error) {
	id, err := context.RunView(&RespondRequestRecipientIdentityView{})
	if err != nil {
		return nil, err
	}
	return id.(view.Identity), nil
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
				Identity:  me,
				AuditInfo: auditInfo,
				Metadata:  metadata,
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

		recipientData := &RecipientData{}
		if err := recipientData.FromBytes(payload); err != nil {
			return nil, err
		}
		if err := ts.WalletManager().RegisterRecipientIdentity(recipientData.Identity, recipientData.AuditInfo, recipientData.Metadata); err != nil {
			return nil, err
		}

		// Update the Endpoint Resolver
		logger.Debugf("bind [%s] to other [%s]", recipientData.Identity, f.Other)
		resolver := view2.GetEndpointService(context)
		err = resolver.Bind(f.Other, recipientData.Identity)
		if err != nil {
			return nil, err
		}

		logger.Debugf("bind me [%s] to [%s]", me, context.Me())
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
	o, err := compileServiceOptions(opts...)
	if err != nil {
		return nil, nil, err
	}
	ids, err := context.RunView(&ExchangeRecipientIdentitiesView{
		TMSID:  o.TMSID(),
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
	if err := ts.WalletManager().RegisterRecipientIdentity(other, request.RecipientData.AuditInfo, request.RecipientData.Metadata); err != nil {
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
		Identity:  me,
		AuditInfo: auditInfo,
		Metadata:  metadata,
	}
	recipientDataRaw, err := recipientData.Bytes()
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
