/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

type RequestApprovalView struct {
	TMSID      token2.TMSID
	TxID       driver.TxID
	RequestRaw []byte
	// RequestAnchor, if not nil it will instruct the approver to verify the token request using this anchor and not the transaction it.
	// This is to be used only for testing.
	RequestAnchor string
	// Nonce, if not nil it will be appended to the messages to sign.
	// This is to be used only for testing.
	Nonce []byte
	// Endorsers are the identities of the FSC node that play the role of endorser
	Endorsers []view.Identity
}

func (r *RequestApprovalView) Call(context view.Context) (interface{}, error) {
	logger.DebugfContext(context.Context(), "request approval...")

	_, tx, err := endorser.NewTransaction(
		context,
		fabric2.WithCreator(r.TxID.Creator),
		fabric2.WithNonce(r.TxID.Nonce),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create endorser transaction")
	}

	tms, err := token2.GetManagementService(context, token2.WithTMSID(r.TMSID))
	if err != nil {
		return nil, errors.WithMessagef(err, "no token management service for [%s]", r.TMSID)
	}
	tx.SetProposal(tms.Namespace(), ChaincodeVersion, InvokeFunction)
	if err := tx.EndorseProposal(); err != nil {
		return nil, errors.WithMessagef(err, "failed to endorse proposal")
	}

	// transient fields
	if err := tx.SetTransientState(TransientTMSIDKey, tms.ID()); err != nil {
		return nil, errors.WithMessagef(err, "failed to set TMS ID transient")
	}
	if err := tx.SetTransient(TransientTokenRequestKey, r.RequestRaw); err != nil {
		return nil, errors.WithMessagef(err, "failed to set token request transient")
	}
	if len(r.RequestAnchor) != 0 {
		if err := tx.SetTransient(TransientRequestAnchorKey, []byte(r.RequestAnchor)); err != nil {
			return nil, errors.WithMessagef(err, "failed to set token request transient")
		}
	}

	logger.DebugfContext(context.Context(), "request endorsement on tx [%s] to [%v]...", tx.ID(), r.Endorsers)
	_, err = context.RunView(endorser.NewParallelCollectEndorsementsOnProposalView(
		tx,
		r.Endorsers...,
	).WithTimeout(2 * time.Minute))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to collect endorsements")
	}
	logger.DebugfContext(context.Context(), "request endorsement done")

	// Return envelope
	env, err := tx.Envelope()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to retrieve envelope for endorsement")
	}
	logger.DebugfContext(context.Context(), "envelope ready")

	return env, nil
}
