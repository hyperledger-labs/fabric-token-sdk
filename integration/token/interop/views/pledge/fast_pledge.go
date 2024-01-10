/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// FastPledgeClaim contains the input information for a pledge+claim
type FastPledgeClaim struct {
	// OriginTMSID identifies the TMS to use to perform the token operation
	OriginTMSID token.TMSID
	// OriginWallet is the identifier of the wallet that owns the tokens to transfer in the origin network
	OriginWallet string
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Issuer is the identity of the issuer's FSC node
	Issuer view.Identity
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// DestinationNetworkURL is the destination network's url to transfer the token to
	DestinationNetworkURL string
	// ReclamationDeadline is the time after which we can reclaim the funds in case they were not transferred
	ReclamationDeadline time.Duration
}

type FastPledgeClaimInitiatorViewFactory struct{}

func (f *FastPledgeClaimInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	v := &FastPledgeClaimInitiatorView{FastPledgeClaim: &FastPledgeClaim{}}
	err := json.Unmarshal(in, v.FastPledgeClaim)
	assert.NoError(err, "failed unmarshalling input")

	return v, nil
}

type FastPledgeClaimInitiatorView struct {
	*FastPledgeClaim
}

func (v *FastPledgeClaimInitiatorView) Call(context view.Context) (interface{}, error) {
	// Collect recipient's token-sdk identity
	recipient, err := pledge.RequestPledgeRecipientIdentity(context, v.Recipient, v.DestinationNetworkURL, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Collect issuer's token-sdk identity
	issuer, err := pledge.RequestRecipientIdentity(context, v.Issuer, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Create a new transaction
	tx, err := pledge.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(v.OriginTMSID),
	)
	assert.NoError(err, "failed created a new transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := pledge.GetWallet(context, v.OriginWallet, token.WithTMSID(v.OriginTMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", v.OriginWallet)

	_, err = tx.Pledge(senderWallet, v.DestinationNetworkURL, v.ReclamationDeadline, recipient, issuer, v.Type, v.Amount)
	assert.NoError(err, "failed pledging")

	// Collect signatures
	_, err = context.RunView(pledge.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign pledge transaction")

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(pledge.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	// Inform the recipient of the pledge,
	// recall that the recipient might be aware of only the other network
	_, err = context.RunView(pledge.NewDistributePledgeInfoView(tx))
	assert.NoError(err, "failed to send the pledge info")

	time.Sleep(v.ReclamationDeadline)

	// Request proof of non-existence for the passed token,
	// we expect the token to exist in the destination network
	w, err := pledge.GetOwnerWallet(context, v.OriginWallet, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed to retrieve wallet of owner during reclaim")

	tokenID := &token2.ID{TxId: tx.ID()}
	_, script, err := pledge.Wallet(context, w).GetPledgedToken(tokenID)
	assert.NoError(err, "failed to retrieve token to be reclaimed")
	assert.False(time.Now().Before(script.Deadline), "cannot reclaim token yet; deadline has not elapsed yet")

	logger.Debugf("request proof of non-existence")
	_, err = pledge.RequestProofOfNonExistence(context, tokenID, v.OriginTMSID, script)
	assert.Error(err, "retrieve proof of non-existence should fail")
	assert.Equal(pledge.TokenExistsError, errors.Cause(err), "token should exist")

	return nil, nil
}

type FastPledgeClaimResponderView struct{}

func (v *FastPledgeClaimResponderView) Call(context view.Context) (interface{}, error) {
	me, err := pledge.RespondRequestPledgeRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the pledge info
	pledgeInfo, err := pledge.ReceivePledgeInfo(context)
	assert.NoError(err, "failed to receive pledge info")

	// Perform any check that is needed to validate the pledge.
	logger.Debugf("The pledge info is %v", pledgeInfo)
	assert.Equal(me, pledgeInfo.Script.Recipient, "recipient is different [%s]!=[%s]", me, pledgeInfo.Script.Recipient)

	// Store the pledge and send a notification back
	_, err = context.RunView(pledge.NewAcceptPledgeIndoView(pledgeInfo))
	assert.NoError(err, "failed accepting pledge info")

	// Retrieve proof of existence of the passed token id
	pledges, err := pledge.Vault(context).PledgeByTokenID(pledgeInfo.TokenID)
	assert.NoError(err, "failed getting pledge")
	assert.Equal(1, len(pledges), "expected one pledge, got [%d]", len(pledges))

	logger.Debugf("request proof of existence")
	proof, err := pledge.RequestProofOfExistence(context, pledges[0])
	assert.NoError(err, "failed to retrieve a valid proof of existence")
	assert.NotNil(proof)

	// Request claim to the issuer
	logger.Debugf("Request claim to the issuer")
	wallet, err := pledge.MyOwnerWallet(context)
	assert.NoError(err, "failed getting my owner wallet")
	me, err = wallet.GetRecipientIdentity()
	assert.NoError(err, "failed getting recipient identity from my owner wallet")

	var session view.Session
	_, err = view2.AsInitiatorCall(context, v, func(context view.Context) (interface{}, error) {
		session, err = pledge.RequestClaim(
			context,
			fabric.GetDefaultIdentityProvider(context).Identity("issuerBeta"), // TODO get issuer
			pledges[0],
			me,
			proof,
		)
		assert.NoError(err, "failed requesting a claim from the issuer")
		return nil, nil
	})
	assert.NoError(err, "failed to request claim")

	return view2.AsResponder(context, session,
		func(context view.Context) (interface{}, error) {
			logger.Debugf("Respond to the issuer...")

			// At some point, the recipient receives the token transaction that in the meantime has been assembled
			tx, err := pledge.ReceiveTransaction(context)
			assert.NoError(err, "failed to receive transaction")

			// The recipient can perform any check on the transaction
			outputs, err := tx.Outputs()
			assert.NoError(err, "failed getting outputs")
			assert.True(outputs.Count() > 0)
			assert.True(outputs.ByRecipient(me).Count() > 0)
			output := outputs.At(0)
			assert.NotNil(output, "failed getting the output")
			assert.NoError(err, "failed parsing quantity")
			assert.Equal(pledges[0].Amount, output.Quantity.ToBigInt().Uint64())
			assert.Equal(pledges[0].TokenType, output.Type)
			assert.Equal(me, output.Owner)

			// If everything is fine, the recipient accepts and sends back her signature.
			_, err = context.RunView(pledge.NewAcceptView(tx))
			assert.NoError(err, "failed to accept the claim transaction")

			// Before completing, the recipient waits for finality of the transaction
			_, err = context.RunView(pledge.NewFinalityView(tx))
			assert.NoError(err, "the claim transaction was not committed")

			// Delete pledges
			err = pledge.Vault(context).Delete(pledges)
			assert.NoError(err, "failed deleting pledges")

			logger.Debugf("Respond to the issuer...done")

			return tx.ID(), nil
		},
	)
}

// FastPledgeReClaim contains the input information for a pledge+reclaim
type FastPledgeReClaim struct {
	// OriginTMSID identifies the TMS to use to perform the token operation
	OriginTMSID token.TMSID
	// OriginWallet is the identifier of the wallet that owns the tokens to transfer in the origin network
	OriginWallet string
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Issuer is the identity of the issuer's FSC node
	Issuer view.Identity
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// DestinationNetworkURL is the destination network's url to transfer the token to
	DestinationNetworkURL string
	// ReclamationDeadline is the time after which we can reclaim the funds in case they were not transferred
	ReclamationDeadline time.Duration
	// PledgeID is the unique identifier of the pledge
	PledgeID string
}

type FastPledgeReClaimInitiatorViewFactory struct{}

func (f *FastPledgeReClaimInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	v := &FastPledgeReClaimInitiatorView{FastPledgeReClaim: &FastPledgeReClaim{}}
	err := json.Unmarshal(in, v.FastPledgeReClaim)
	assert.NoError(err, "failed unmarshalling input")

	return v, nil
}

type FastPledgeReClaimInitiatorView struct {
	*FastPledgeReClaim
}

func (v *FastPledgeReClaimInitiatorView) Call(context view.Context) (interface{}, error) {
	// Collect recipient's token-sdk identity
	recipient, err := pledge.RequestPledgeRecipientIdentity(context, v.Recipient, v.DestinationNetworkURL, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Collect issuer's token-sdk identity
	issuer, err := pledge.RequestRecipientIdentity(context, v.Issuer, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Create a new transaction
	tx, err := pledge.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(v.OriginTMSID),
	)
	assert.NoError(err, "failed created a new transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := pledge.GetWallet(context, v.OriginWallet, token.WithTMSID(v.OriginTMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", v.OriginWallet)

	_, err = tx.Pledge(senderWallet, v.DestinationNetworkURL, v.ReclamationDeadline, recipient, issuer, v.Type, v.Amount)
	assert.NoError(err, "failed pledging")

	// Collect signatures
	_, err = context.RunView(pledge.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign pledge transaction")

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(pledge.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	// Inform the recipient of the pledge,
	// recall that the recipient might be aware of only the other network
	_, err = context.RunView(pledge.NewDistributePledgeInfoView(tx))
	assert.NoError(err, "failed to send the pledge info")

	time.Sleep(v.ReclamationDeadline)

	// Reclaim, here we are executing the reclaim protocol, therefore
	// we initiate it with a fresh context
	tokenID := &token2.ID{TxId: tx.ID()}
	_, err = view2.Initiate(context, &ReclaimInitiatorView{
		&Reclaim{
			TMSID:    v.OriginTMSID,
			TokenID:  tokenID,
			WalletID: v.OriginWallet,
			Retry:    false,
		},
	})
	assert.NoError(err, "failed to reclaim [%s:%s:%s]", tokenID, v.OriginTMSID, v.OriginWallet)

	return nil, nil
}

type FastPledgeReClaimResponderView struct{}

func (v *FastPledgeReClaimResponderView) Call(context view.Context) (interface{}, error) {
	me, err := pledge.RespondRequestPledgeRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the pledge info
	pledgeInfo, err := pledge.ReceivePledgeInfo(context)
	assert.NoError(err, "failed to receive pledge info")

	// Perform any check that is needed to validate the pledge.
	logger.Debugf("The pledge info is %v", pledgeInfo)
	assert.Equal(me, pledgeInfo.Script.Recipient, "recipient is different [%s]!=[%s]", me, pledgeInfo.Script.Recipient)

	// Store the pledge and send a notification back
	_, err = context.RunView(pledge.NewAcceptPledgeIndoView(pledgeInfo))
	assert.NoError(err, "failed accepting pledge info")

	// Retrieve proof of existence of the passed token id
	pledges, err := pledge.Vault(context).PledgeByTokenID(pledgeInfo.TokenID)
	assert.NoError(err, "failed getting pledge")
	assert.Equal(1, len(pledges), "expected one pledge, got [%d]", len(pledges))

	logger.Debugf("request proof of existence")
	proof, err := pledge.RequestProofOfExistence(context, pledges[0])
	assert.NoError(err, "failed to retrieve a valid proof of existence")
	assert.NotNil(proof)

	// Don't claim the token
	return nil, nil
}
