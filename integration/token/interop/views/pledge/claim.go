/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Claim contains the input information to claim a token
type Claim struct {
	// OriginTokenID is the identifier of the pledged token in the origin network
	OriginTokenID *token.ID
	// Issuer is the identity of the issuer in the destination network
	Issuer string
}

// ClaimInitiatorView is the view of the initiator of the claim (Bob)
type ClaimInitiatorView struct {
	*Claim
}

func (c *ClaimInitiatorView) Call(context view.Context) (interface{}, error) {
	// Retrieve proof of existence of the passed token id
	pledges, err := pledge.Vault(context).PledgeByTokenID(c.OriginTokenID)
	assert.NoError(err, "failed getting pledge")
	assert.Equal(1, len(pledges), "expected one pledge, got [%d]", len(pledges))

	logger.Debugf("request proof of existence")
	proof, err := pledge.RequestProofOfExistence(context, pledges[0])
	assert.NoError(err, "failed to retrieve a valid proof of existence")
	assert.NotNil(proof)

	// Request claim to the issuer
	wallet, err := pledge.MyOwnerWallet(context)
	assert.NoError(err, "failed getting my owner wallet")
	me, err := wallet.GetRecipientIdentity()
	assert.NoError(err, "failed getting recipient identity from my owner wallet")

	// Contact the issuer, present the pledge proof, and ask to initiate the issue process
	session, err := pledge.RequestClaim(
		context,
		fabric.GetDefaultIdentityProvider(context).Identity(c.Issuer),
		pledges[0],
		me,
		proof,
	)
	assert.NoError(err, "failed requesting a claim from the issuer")

	// Now we have an inversion of roles.
	// The issuer becomes the initiator of the issue transaction,
	// and this node is the responder
	return view2.AsResponder(context, session,
		func(context view.Context) (interface{}, error) {
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

			return tx.ID(), nil
		},
	)
}

type ClaimInitiatorViewFactory struct{}

func (c *ClaimInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	f := &ClaimInitiatorView{Claim: &Claim{}}
	err := json.Unmarshal(in, f.Claim)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

// ClaimIssuerView is the view of the issuer responding to the claim interactive protocol.
type ClaimIssuerView struct{}

func (c *ClaimIssuerView) Call(context view.Context) (interface{}, error) {
	// Receive claim request
	req, err := pledge.ReceiveClaimRequest(context)
	assert.NoError(err, "failed to receive claim request")

	// Validate and check Proof
	err = pledge.ValidateClaimRequest(context, req)
	assert.NoError(err, "failed validating claim request")
	logger.Debugf("claim request valid, preparing claim transaction [%s]", err)

	// At this point, the issuer is ready to prepare the token transaction.
	tx, err := pledge.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating a new transaction")

	// The issuer adds a new claim operation to the transaction following the instruction received.
	wallet, err := pledge.GetIssuerWallet(context, "")
	assert.NoError(err, "failed to get issuer's wallet")

	err = tx.Claim(wallet, req.TokenType, req.Quantity, req.Recipient, req.OriginTokenID, req.OriginNetwork, req.PledgeProof)
	assert.NoError(err, "failed adding a claim action")

	// The issuer is ready to collect all the required signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	_, err = context.RunView(pledge.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements on claim transaction")

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(pledge.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit claim transaction")

	return tx.ID(), nil
}
