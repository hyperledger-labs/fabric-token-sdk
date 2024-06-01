/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.asset.transfer")

// Reclaim contains the input information for a reclaim
type Reclaim struct {
	TMSID token2.TMSID
	// TokenID contains the identifier of the token to be reclaimed.
	TokenID *token.ID
	// WalletID is the identifier of the wallet that the tokens will be reclaimed to
	WalletID string
	// Retry tells if a retry must happen in case of a failure
	Retry bool
}

type ReclaimInitiatorView struct {
	*Reclaim
}

func (rv *ReclaimInitiatorView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("caller [%s]", context.Initiator())
	// Request proof of non-existence for the passed token
	w, err := pledge.GetOwnerWallet(context, rv.WalletID, token2.WithTMSID(rv.TMSID))
	assert.NoError(err, "failed to retrieve wallet of owner during reclaim")

	token, script, err := pledge.Wallet(context, w).GetPledgedToken(rv.TokenID)
	assert.NoError(err, "failed to retrieve token to be reclaimed")
	if time.Now().Before(script.Deadline) {
		return nil, errors.Errorf("cannot reclaim token yet; deadline has not elapsed yet")
	}

	logger.Debugf("request proof of non-existence")
	proof, err := pledge.RequestProofOfNonExistence(context, rv.TokenID, rv.TMSID, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve proof of non-existence")
	}

	// At this point, Alice contacts the issuer's FSC node
	// to ask for the issuer's signature on the TokenID
	issuerSignature, err := pledge.RequestIssuerSignature(context, rv.TokenID, rv.TMSID, script, proof)
	assert.NoError(err, "failed getting issuer's signature")
	assert.NotNil(issuerSignature)

	// At this point, alice is ready to prepare the token transaction.
	tx, err := pledge.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(rv.TMSID),
	)
	assert.NoError(err, "failed to create a new transaction")

	// Create reclaim transaction
	err = tx.Reclaim(w, token, issuerSignature, rv.TokenID, proof)
	assert.NoError(err, "failed creating transaction")

	// Alice is ready to collect all the required signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(pledge.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for finality
	_, err = context.RunView(pledge.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type ReclaimViewFactory struct{}

func (rvf *ReclaimViewFactory) NewView(in []byte) (view.View, error) {
	rv := &ReclaimInitiatorView{Reclaim: &Reclaim{}}
	err := json.Unmarshal(in, rv.Reclaim)
	assert.NoError(err, "failed unmarshalling input")
	return rv, nil
}

type ReclaimIssuerResponderView struct {
	WalletID string
	Network  string
}

func (i *ReclaimIssuerResponderView) Call(context view.Context) (interface{}, error) {
	_, err := pledge.RespondRequestIssuerSignature(context, i.WalletID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to respond to signature request")
	}

	return nil, nil
}
