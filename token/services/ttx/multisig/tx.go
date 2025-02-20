/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Transaction struct {
	*ttx.Transaction
}

func Wrap(tx *ttx.Transaction) *Transaction {
	return &Transaction{Transaction: tx}
}

func (t *Transaction) Lock(senderWallet *token2.OwnerWallet, tokenType token.Type, amount uint64, recipients []token2.Identity, opts ...token2.TransferOption) error {
	raw, err := multisig.Wrap(recipients...)
	if err != nil {
		return errors.Wrap(err, "failed wrapping identities")
	}
	return t.Transaction.Transfer(
		senderWallet,
		tokenType,
		[]uint64{amount},
		[]token2.Identity{raw},
	)
}

func (t *Transaction) Spend(wallet *token2.OwnerWallet, at *token.UnspentToken, recipient token2.Identity) error {
	panic("implement me")
}
