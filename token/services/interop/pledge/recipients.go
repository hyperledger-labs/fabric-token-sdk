/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

type RecipientData = token.RecipientData

type RecipientRequest struct {
	NetworkURL string
	WalletID   []byte
}

type RequestRecipientIdentityView struct {
	TMSID       token.TMSID
	DestNetwork string
	Other       view.Identity
}

// RequestPledgeRecipientIdentity executes the RequestRecipientIdentityView.
// The sender contacts the recipient's FSC node identified via the passed view identity.
// The sender gets back the identity the recipient wants to use to assign ownership of tokens.
func RequestPledgeRecipientIdentity(context view.Context, recipient view.Identity, destNetwork string, opts ...token.ServiceOption) (view.Identity, error) {
	options, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}
	pseudonymBoxed, err := context.RunView(&RequestRecipientIdentityView{
		TMSID:       options.TMSID(),
		DestNetwork: destNetwork,
		Other:       recipient,
	})
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
		session, err := session2.NewJSON(context, context.Initiator(), f.Other)
		if err != nil {
			return nil, err
		}

		// Ask for identity
		err = session.Send(&RecipientRequest{
			NetworkURL: f.DestNetwork,
			WalletID:   f.Other,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to send recipient request")
		}

		// Wait to receive a view identity
		recipientData := &RecipientData{}
		if err := session.Receive(recipientData); err != nil {
			return nil, errors.Wrapf(err, "failed to receive recipient data")
		}
		//if err := tms.WalletManager().RegisterRecipientIdentity(recipientData.Identity, recipientData.AuditInfo, recipientData.Metadata); err != nil {
		//	return nil, err
		//}

		// Update the Endpoint Resolver
		if err := view2.GetEndpointService(context).Bind(f.Other, recipientData.Identity); err != nil {
			return nil, err
		}

		return recipientData.Identity, nil
	}
}

type RespondRequestPledgeRecipientIdentityView struct {
	Wallet string
}

// RespondRequestPledgeRecipientIdentity executes the RespondRequestPledgeRecipientIdentityView.
// The recipient sends back the identity to receive ownership of tokens.
// The identity is taken from the wallet
func RespondRequestPledgeRecipientIdentity(context view.Context) (view.Identity, error) {
	id, err := context.RunView(&RespondRequestPledgeRecipientIdentityView{})
	if err != nil {
		return nil, err
	}
	return id.(view.Identity), nil
}

func (s *RespondRequestPledgeRecipientIdentityView) Call(context view.Context) (interface{}, error) {
	session := session2.JSON(context)
	recipientRequest := &RecipientRequest{}
	if err := session.Receive(recipientRequest); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling recipient request")
	}

	wallet := s.Wallet
	if len(wallet) == 0 && len(recipientRequest.WalletID) != 0 {
		wallet = string(recipientRequest.WalletID)
	}

	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.Errorf("failed to load state service provider")
	}
	tmsID, err := ssp.URLToTMSID(recipientRequest.NetworkURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing destination [%s]", recipientRequest.NetworkURL)
	}
	w := GetWallet(
		context,
		wallet,
		token.WithTMSID(tmsID),
	)
	if w == nil {
		return nil, errors.Errorf("unable to get wallet %s in %s", wallet, tmsID)
	}
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

	// Step 3: send the public key back to the invoker
	err = session.Send(&RecipientData{
		Identity:  recipientIdentity,
		AuditInfo: auditInfo,
		Metadata:  metadata,
	})
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

// RequestRecipientIdentity executes the RequestRecipientIdentityView.
func RequestRecipientIdentity(context view.Context, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, error) {
	return ttx.RequestRecipientIdentity(context, recipient, opts...)
}

// RespondRequestRecipientIdentity executes the RespondRequestRecipientIdentityView.
func RespondRequestRecipientIdentity(context view.Context) (view.Identity, error) {
	return ttx.RespondRequestRecipientIdentity(context)
}
