/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutils

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const defaultCurrency = "CHF"

var (
	logger                    = flogging.MustGetLogger("token-sdk.selector.tests")
	defaultWalletOwner        = &token.Owner{Raw: []byte{1, 2, 3}}
	defaultTokenFilter        = &TokenFilter{Wallet: defaultWalletOwner}
	txId               uint32 = 0
)

type EnhancedManager interface {
	token2.SelectorManager
	TokenSum() (token.Quantity, error)
	UpdateTokens(spentTokens []*token.ID, addedTokens []token.UnspentToken) error
}

func TestSufficientTokensOneReplica(t *testing.T, replica EnhancedManager) {
	// Create 2 tokens of value CHF1 each (total CHF2)
	item := newToken(1)
	unspentTokens := createDefaultTokens(collections.Repeat(item, 2)...)
	err := storeTokens(replica, unspentTokens)
	assert.NoError(t, err)

	// The replica asks for CHF1
	errs := parallelSelect(t, []EnhancedManager{replica}, []token.Quantity{newToken(1)})
	assert.Empty(t, errs)
}

func TestSufficientTokensBigDenominationsOneReplica(t *testing.T, replica EnhancedManager) {
	// Create 1 token of value CHF100
	unspentTokens := createDefaultTokens(newToken(100))
	err := storeTokens(replica, unspentTokens)
	assert.NoError(t, err)

	// The replica asks for CHF1, CHF1, ..., CHF1 for 100 times (total CHF100)
	item := newToken(1)
	errs := parallelSelect(t, []EnhancedManager{replica}, collections.Repeat(item, 100))
	assert.Empty(t, errs)
}

func TestSufficientTokensBigDenominationsManyReplicas(t *testing.T, replicas []EnhancedManager) {
	// Create 2 tokens of value CHF150 each (total CHF300)
	item := newToken(150)
	unspentTokens := createDefaultTokens(collections.Repeat(item, 2)...)
	err := storeTokens(replicas[0], unspentTokens)
	assert.NoError(t, err)

	// Each replica asks for CHF100 (total CHF300)
	item = newToken(1)
	errs := parallelSelect(t, replicas, collections.Repeat(item, 100))
	assert.Empty(t, errs)
}

func TestInsufficientTokensOneReplica(t *testing.T, replica EnhancedManager) {
	// Create 2 tokens of value CHF1 each (total CHF2)
	item := newToken(1)
	unspentTokens := createDefaultTokens(collections.Repeat(item, 2)...)
	err := storeTokens(replica, unspentTokens)
	assert.NoError(t, err)

	// The replica asks for CHF1, CHF1, CHF1 (total CHF3)
	item = newToken(1)
	errs := parallelSelect(t, []EnhancedManager{replica}, collections.Repeat(item, 3))
	assert.Len(t, errs, 1)
}

func TestSufficientTokensManyReplicas(t *testing.T, replicas []EnhancedManager) {
	// Create 100 tokens of value CHF1 each (total CHF100)
	item := newToken(1)
	unspentTokens := createDefaultTokens(collections.Repeat(item, 100)...)
	err := storeTokens(replicas[0], unspentTokens)
	assert.NoError(t, err)

	// Each replica asks for CHF5 (total CHF100)
	errs := parallelSelect(t, replicas, []token.Quantity{newToken(5)})
	assert.Empty(t, errs)
}

func TestInsufficientTokensManyReplicas(t *testing.T, replicas []EnhancedManager) {
	// Create 100 tokens of value CHF2 each (total CHF200)
	item := newToken(2)
	unspentTokens := createDefaultTokens(collections.Repeat(item, 50)...)
	err := storeTokens(replicas[0], unspentTokens)
	assert.NoError(t, err)

	// Each replica asks for CHF3, CHF3, CHF3, and CHF3 (total CHF 240)
	item = newToken(3)
	errs := parallelSelect(t, replicas, collections.Repeat(item, 4))
	assert.NotEmpty(t, errs)
	sum, err := replicas[0].TokenSum()
	assert.NoError(t, err)
	assert.Equal(t, 0, sum.Cmp(newToken(1)))
}

// Enhanced manager

type enhancedManager struct {
	token2.SelectorManager
	tokenDB driver.TokenDB
}

func NewEnhancedManager(manager token2.SelectorManager, tokenDB driver.TokenDB) *enhancedManager {
	return &enhancedManager{
		SelectorManager: manager,
		tokenDB:         tokenDB,
	}
}

func (m *enhancedManager) TokenSum() (token.Quantity, error) {
	unspent, err := m.tokenDB.ListUnspentTokens()
	if err != nil {
		return nil, err
	}
	sum := unspent.Sum(TokenQuantityPrecision)
	return sum, nil
}

func (m *enhancedManager) UpdateTokens(deleted []*token.ID, added []token.UnspentToken) error {
	tx, err := m.tokenDB.NewTokenDBTransaction(context.TODO())
	if err != nil {
		return err
	}
	if len(deleted) > 0 {
		for _, t := range deleted {
			if err := tx.Delete(context.TODO(), t.TxId, t.Index, "me"); err != nil {
				err2 := tx.Rollback()
				return errors.Wrapf(err, "failed to delete - while rolling back: %v", err2)
			}
		}
	}
	if len(added) > 0 {
		for _, t := range added {
			if err := tx.StoreToken(context.TODO(), driver.TokenRecord{
				TxID:           t.Id.TxId,
				Index:          t.Id.Index,
				IssuerRaw:      []byte{},
				OwnerRaw:       t.Owner.Raw,
				OwnerType:      "idemix",
				OwnerIdentity:  []byte{},
				Ledger:         []byte("ledger"),
				LedgerMetadata: []byte{},
				Quantity:       t.Quantity,
				Type:           t.Type,
				Amount:         0,
				Owner:          true,
				Auditor:        false,
				Issuer:         false,
			}, []string{"alice"}); err != nil {
				err2 := tx.Rollback()
				return errors.Wrapf(err, "failed to insert - while rolling back: %v", err2)
			}
		}
	}
	return tx.Commit()
}

// Utils

func newTxID() string {
	return fmt.Sprintf("tx%d", atomic.AddUint32(&txId, 1))
}

func parallelSelect(t *testing.T, replicas []EnhancedManager, quantities []token.Quantity) []error {
	errCh := make(chan error, 100)
	errs := make([]error, 0)
	go func() {
		for err := range errCh {
			errs = append(errs, err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(len(quantities) * len(replicas))
	for _, replica := range replicas {
		for _, quantity := range quantities {
			quantity, replica := quantity, replica
			txID := newTxID()
			sel, err := replica.NewSelector(txID)
			assert.NoError(t, err)
			go func() {
				defer replica.Close(txID)
				tokens, sum, err := sel.Select(defaultTokenFilter, quantity.Hex(), defaultCurrency)
				if err != nil {
					errCh <- err
				} else {
					assert.NotNil(t, sum)
					change := sum.Sub(quantity)
					assert.GreaterOrEqual(t, change.ToBigInt().Int64(), int64(0))
					assert.Greater(t, len(tokens), 0)
					assert.NoError(t, deleteTokensAndStoreChange(replica, tokens, change))
				}
				if tokenSum, err := replica.TokenSum(); err == nil {
					logger.Infof("Current sum of tokens in the DB: %s", tokenSum.Decimal())
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()
	close(errCh)
	return errs
}

func storeTokens(m EnhancedManager, added []token.UnspentToken) error {
	return m.UpdateTokens(nil, added)
}

func deleteTokensAndStoreChange(m EnhancedManager, spentTokens []*token.ID, change token.Quantity) error {
	logger.Debugf("Deleting [%d] tokens [%s] and creating a new one with quantity %s", len(spentTokens), spentTokens, change.Decimal())
	var changeTokens []token.UnspentToken
	if change.ToBigInt().Int64() > 0 {
		changeTokens = createTokens(map[transaction.ID][]token.Quantity{
			newTxID(): {change},
		})
	}
	for {
		if err := m.UpdateTokens(spentTokens, changeTokens); err == nil {
			return nil
		} else {
			logger.Warnf("Failed to delete tokens: %v. Retrying", err)
		}
	}
}

func createDefaultTokens(quantities ...token.Quantity) []token.UnspentToken {
	return createTokens(map[transaction.ID][]token.Quantity{newTxID(): quantities})
}

func createTokens(txs map[transaction.ID][]token.Quantity) []token.UnspentToken {
	unspentTokens := make([]token.UnspentToken, 0)
	for txID, quantities := range txs {
		for i, quantity := range quantities {
			unspentTokens = append(unspentTokens, token.UnspentToken{
				Id:       &token.ID{TxId: txID, Index: uint64(i)},
				Owner:    defaultWalletOwner,
				Type:     defaultCurrency,
				Quantity: quantity.Hex(),
			})
		}
	}
	return unspentTokens
}

func newToken(quantity int) token.Quantity {
	return token.NewQuantityFromUInt64(uint64(quantity))
}
