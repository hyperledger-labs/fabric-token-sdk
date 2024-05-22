/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenIterator interface {
	UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error)
	UnlockIDs(tokenIDs ...*token2.ID) []*token2.ID
}

type SimpleSelector struct {
	TxID          string
	QuerySelector TokenIterator
	Precision     uint64
	TokenIDs      []*token2.ID
}

func (n *SimpleSelector) Select(ownerFilter token.OwnerFilter, q, tokenType string) ([]*token2.ID, token2.Quantity, error) {
	logger.Debugf("call selector for ... %v %v %v", ownerFilter, q, tokenType)
	iter, err := n.QuerySelector.UnspentTokensIteratorBy(ownerFilter.ID(), tokenType)
	if err != nil {
		return nil, nil, err
	}

	sum := token2.NewZeroQuantity(n.Precision)

	// TODO can we make the q already a quantity?
	// that would require to change the selector API
	target, err := token2.ToQuantity(q, n.Precision)
	if err != nil {
		return nil, nil, err
	}

	var tokens []*token2.ID
	for {
		next, err := iter.Next()
		if err != nil {
			// should we retry again? check message
			continue
		}
		if next == nil {
			// no more tokens
			break
		}
		q, err := token2.ToQuantity(next.Quantity, n.Precision)
		if err != nil {
			n.QuerySelector.UnlockIDs(tokens...)
			return nil, nil, errors.Wrap(err, "failed to convert quantity")
		}

		sum = sum.Add(q)
		tokens = append(tokens, next.Id)

		if target.Cmp(sum) <= 0 {
			break
		}
	}

	if target.Cmp(sum) <= 0 {
		n.TokenIDs = append(n.TokenIDs, tokens...)
		logger.Debugf("selector returns tokens=%v", tokens)
		return tokens, sum, nil
	}

	// release the selected tokens immediately and return error
	n.QuerySelector.UnlockIDs(tokens...)
	err = errors.WithMessagef(
		token.SelectorInsufficientFunds,
		"token selection failed: insufficient funds, only [%s] tokens of type [%s] are available", sum.Decimal(), tokenType,
	)
	logger.Debugf("selector returns with error: [%s]", err)

	return nil, nil, err
}
