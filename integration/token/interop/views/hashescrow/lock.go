/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"crypto"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Lock struct {
	TMSID         token.TMSID
	Wallet        string
	Type          token2.Type
	Amount        uint64
	Recipient     view.Identity
	RecipientHash []byte
	SenderHash    []byte
	HashFunc      crypto.Hash
}

type LockInfo struct {
	TxID          string
	RecipientHash []byte
	SenderHash    []byte
}

type LockView struct {
	*Lock
}

func (hv *LockView) Call(context view.Context) (res any, err error) {
	var tx *hashescrow.Transaction
	defer func() {
		if e := recover(); e != nil {
			txID := "none"
			if tx != nil {
				txID = tx.ID()
			}
			if err == nil {
				err = errors.Errorf("<<<[%s]>>>: %s", txID, e)
			} else {
				err = errors.Errorf("<<<[%s]>>>: %s", txID, err)
			}
		}
	}()

	me, recipient, err := ttx.ExchangeRecipientIdentities(context, "", hv.Recipient, token.WithTMSID(hv.TMSID))
	assert.NoError(err, "failed getting recipient identity")

	idProvider, err := id.GetProvider(context)
	assert.NoError(err, "failed getting id provider")
	tx, err = hashescrow.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(idProvider.Identity("auditor")),
		ttx.WithTMSID(hv.TMSID),
	)
	assert.NoError(err, "failed creating a hash escrow transaction")

	senderWallet := hashescrow.GetWallet(context, hv.Wallet, token.WithTMSID(hv.TMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", hv.Wallet)

	_, err = tx.Lock(
		context.Context(),
		senderWallet,
		me,
		hv.Type,
		hv.Amount,
		recipient,
		hashescrow.WithRecipientHash(hv.RecipientHash),
		hashescrow.WithSenderHash(hv.SenderHash),
		hashescrow.WithHashFunc(hv.HashFunc),
	)
	assert.NoError(err, "failed adding a hash escrow lock operation")

	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx.Transaction))
	assert.NoError(err, "failed to collect endorsements for hash escrow transaction")

	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx.Transaction))
	assert.NoError(err, "failed to commit hash escrow transaction")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	var script *hashescrow.Script
	for i := range outputs.Count() {
		output, err := hashescrow.ToOutput(outputs.At(i))
		assert.NoError(err, "cannot get hash escrow output wrapper")
		if !output.IsHashEscrow() {
			continue
		}
		script, err = output.Script()
		assert.NoError(err, "cannot get hash escrow script from output")

		break
	}
	assert.NotNil(script, "expected a hash escrow script output")

	return &LockInfo{
		TxID:          tx.ID(),
		RecipientHash: script.RecipientHashInfo.Hash,
		SenderHash:    script.SenderHashInfo.Hash,
	}, nil
}

type LockViewFactory struct{}

func (p *LockViewFactory) NewView(in []byte) (view.View, error) {
	f := &LockView{Lock: &Lock{}}
	err := json.Unmarshal(in, f.Lock)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type LockAcceptView struct{}

func (h *LockAcceptView) Call(context view.Context) (any, error) {
	_, _, err := ttx.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed to respond to identity request")

	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.True(outputs.Count() >= 1, "expected at least one output, got [%d]", outputs.Count())

	_, err = context.RunView(ttx.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return tx, nil
}
