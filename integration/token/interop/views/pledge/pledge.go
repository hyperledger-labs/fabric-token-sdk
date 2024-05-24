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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type Result struct {
	TxID     string
	PledgeID string
}

// Pledge contains the input information for a transfer
type Pledge struct {
	// OriginTMSID identifies the TMS to use to perform the token operation.
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

// View is the view of the initiator of a pledge operation
type View struct {
	*Pledge
}

func (v *View) Call(context view.Context) (interface{}, error) {
	// Collect recipient's token-sdk identity
	recipient, err := pledge.RequestPledgeRecipientIdentity(context, v.Recipient, v.DestinationNetworkURL, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Collect issuer's token-sdk identity
	// TODO: shall we ask for the issuer identity here and not the owner identity?
	issuer, err := pledge.RequestRecipientIdentity(context, v.Issuer, token.WithTMSID(v.OriginTMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Create a new transaction
	tx, err := pledge.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(v.OriginTMSID),
	)
	assert.NoError(err, "failed created a new transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := pledge.GetWallet(context, v.OriginWallet, token.WithTMSID(v.OriginTMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", v.OriginWallet)

	pledgeID, err := tx.Pledge(senderWallet, v.DestinationNetworkURL, v.ReclamationDeadline, recipient, issuer, v.Type, v.Amount)
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

	return json.Marshal(&Result{TxID: tx.ID(), PledgeID: pledgeID})
}

type ViewFactory struct{}

func (f *ViewFactory) NewView(in []byte) (view.View, error) {
	v := &View{Pledge: &Pledge{}}
	err := json.Unmarshal(in, v.Pledge)
	assert.NoError(err, "failed unmarshalling input")

	return v, nil
}

type RecipientResponderView struct{}

func (p *RecipientResponderView) Call(context view.Context) (interface{}, error) {
	me, err := pledge.RespondRequestPledgeRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the pledge info
	pledgeInfo, err := pledge.ReceivePledgeInfo(context)
	assert.NoError(err, "failed to receive pledge info")

	// Perform any check that is needed to validate the pledge.
	logger.Debugf("The pledge info is %v", pledgeInfo)
	assert.Equal(me, pledgeInfo.Script.Recipient, "recipient is different [%s]!=[%s]", me, pledgeInfo.Script.Recipient)

	// TODO: check pledgeInfo.Script.DestinationNetwork

	// Store the pledge and send a notification back
	_, err = context.RunView(pledge.NewAcceptPledgeIndoView(pledgeInfo))
	assert.NoError(err, "failed accepting pledge info")

	return nil, nil
}

type IssuerResponderView struct{}

func (p *IssuerResponderView) Call(context view.Context) (interface{}, error) {
	me, err := pledge.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the token transaction that in the meantime has been assembled
	tx, err := pledge.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.True(outputs.Count() >= 1, "expected at least one output, got [%d]", outputs.Count())
	outputs = outputs.ByScript()
	assert.True(outputs.Count() == 1, "expected only one pledge output, got [%d]", outputs.Count())
	script := outputs.ScriptAt(0)
	assert.NotNil(script, "expected a pledge script")
	assert.Equal(me, script.Issuer, "Expected pledge script to have me (%x) as an issuer but it has %x instead", me, script.Issuer)

	// If everything is fine, the recipient accepts and sends back her signature.
	// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
	// This depends on the driver implementation.
	_, err = context.RunView(pledge.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// The issue is in the same Fabric network of the pledge
	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(pledge.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return nil, nil
}
