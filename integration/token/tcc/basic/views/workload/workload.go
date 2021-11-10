/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package workload

import (
	"encoding/json"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("workload")

type AcceptCashView struct{}

func (a *AcceptCashView) Call(context view.Context) (interface{}, error) {
	logger.Infof("accept new tokens [%s]", context.ID())

	_, err := ttxcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	tx, err := ttxcc.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	_, err = context.RunView(ttxcc.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	return nil, nil
}

type TransferWorkload struct {
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet string
	// TokenIDs contains a list of token ids to transfer. If empty, tokens are selected on the spot.
	TokenIDs []*token.ID
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity

	NumBatches int
	BatchSize  int
}

type TransferWorkloadView struct {
	*TransferWorkload
}

func (t *TransferWorkloadView) Call(context view.Context) (interface{}, error) {
	var batches [][]*ttxcc.Transaction

	for i := 0; i < t.NumBatches; i++ {
		var batch []*ttxcc.Transaction
		for j := 0; j < t.BatchSize; j++ {
			assert.NoError(context.ResetSessions(), "failed resetting sessions")

			recipient, err := ttxcc.RequestRecipientIdentity(context, t.Recipient)
			assert.NoError(err, "failed getting recipient")

			tx, err := ttxcc.NewAnonymousTransaction(
				context,
				ttxcc.WithAuditor(fabric.GetDefaultIdentityProvider(context).Identity("auditor")),
			)
			assert.NoError(err, "failed creating transaction")

			senderWallet := ttxcc.GetWallet(context, t.Wallet)
			assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

			err = tx.Transfer(
				senderWallet,
				t.Type,
				[]uint64{t.Amount},
				[]view.Identity{recipient},
				token2.WithTokenIDs(t.TokenIDs...),
			)
			assert.NoError(err, "failed adding new tokens")

			_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
			assert.NoError(err, "failed to sign transaction")

			batch = append(batch, tx)
		}
		batches = append(batches, batch)
	}

	var wg sync.WaitGroup
	wg.Add(len(batches))
	for _, batch := range batches {
		go func(transactions []*ttxcc.Transaction) {
			defer wg.Done()
			t.sendBatch(context, transactions)
		}(batch)
	}
	wg.Wait()

	return nil, nil
}

func (t *TransferWorkloadView) sendBatch(context view.Context, transactions []*ttxcc.Transaction) {
	for _, tx := range transactions {
		// Send to the ordering service and wait for finality
		_, err := context.RunView(ttxcc.NewOrderingView(tx))
		assert.NoError(err, "failed asking ordering")
	}
	for _, tx := range transactions {
		// Send to the ordering service and wait for finality
		_, err := context.RunView(ttxcc.NewFinalityView(tx))
		assert.NoError(err, "failed asking ordering")
	}
}

type TransferWorkloadViewFactory struct{}

func (p *TransferWorkloadViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferWorkloadView{TransferWorkload: &TransferWorkload{}}
	err := json.Unmarshal(in, f.TransferWorkload)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type IssueWorkload struct {
	// IssuerWallet is the issuer's wallet to use
	IssuerWallet string
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represent the number of units of a certain token type stored in the token
	Quantity uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity

	NumBatches int
	BatchSize  int
}

type IssueWorkloadView struct {
	*IssueWorkload
}

func (t *IssueWorkloadView) Call(context view.Context) (interface{}, error) {
	logger.Infof("issue workload [%d,%d]", t.NumBatches, t.BatchSize)

	batches := make([][]*ttxcc.Transaction, t.NumBatches)

	var wg sync.WaitGroup
	wg.Add(len(batches))
	for i := 0; i < t.NumBatches; i++ {
		go func(index int) {
			defer wg.Done()
			batches[index] = t.prepareBatch(index, context)
		}(i)
	}
	wg.Wait()

	wg.Add(len(batches))
	for _, batch := range batches {
		go func(transactions []*ttxcc.Transaction) {
			defer wg.Done()
			t.sendBatch(context, transactions)
		}(batch)
	}
	wg.Wait()

	return nil, nil
}

func (t *IssueWorkloadView) prepareBatch(i int, context view.Context) []*ttxcc.Transaction {
	viewManager := view2.GetManager(context)
	assert.NotNil(viewManager, "view manager should not be nil")

	var batch []*ttxcc.Transaction
	for j := 0; j < t.BatchSize; j++ {
		logger.Infof("prepare issue transaction [%d] for batch [%d]", j, i)

		txBoxed, err := viewManager.InitiateView(&IssueView{&Issue{
			IssuerWallet: t.IssuerWallet,
			TokenType:    t.TokenType,
			Quantity:     t.Quantity,
			Recipient:    t.Recipient,
		}})
		assert.NoError(err, "failed getting recipient identity")

		batch = append(batch, txBoxed.(*ttxcc.Transaction))
	}
	return batch
}

func (t *IssueWorkloadView) sendBatch(context view.Context, transactions []*ttxcc.Transaction) {
	for _, tx := range transactions {
		// Send to the ordering service and wait for finality
		_, err := context.RunView(ttxcc.NewOrderingView(tx))
		assert.NoError(err, "failed asking ordering")
	}
	for _, tx := range transactions {
		// Send to the ordering service and wait for finality
		_, err := context.RunView(ttxcc.NewFinalityView(tx))
		assert.NoError(err, "failed asking ordering")
	}
}

type IssueWorkloadViewFactory struct{}

func (p *IssueWorkloadViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssueWorkloadView{IssueWorkload: &IssueWorkload{}}
	err := json.Unmarshal(in, f.IssueWorkload)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
