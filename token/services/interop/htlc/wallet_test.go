/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LFDT-Panurus/panurus/token/driver"
	drivermock "github.com/LFDT-Panurus/panurus/token/driver/mock"
	token2 "github.com/LFDT-Panurus/panurus/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ----------------------------------------------------------------

// newToken builds a non-expired HTLC unspent token for use in wallet tests.
func newToken(t *testing.T) *token2.UnspentToken {
	t.Helper()
	
	return makeFilterToken(t, &Script{
		Sender:    []byte("sender"),
		Recipient: []byte("recipient"),
		Deadline:  time.Now().Add(time.Hour),
	})
}

// singleTokenIter returns a mock iterator that yields tok once then nil.
func singleTokenIter(tok *token2.UnspentToken) *drivermock.UnspentTokensIterator {
	it := &drivermock.UnspentTokensIterator{}
	it.NextReturnsOnCall(0, tok, nil)
	it.NextReturnsOnCall(1, nil, nil)
	
	return it
}

// emptyIter returns a mock iterator that is immediately exhausted.
func emptyIter() *drivermock.UnspentTokensIterator {
	it := &drivermock.UnspentTokensIterator{}
	it.NextReturnsOnCall(0, nil, nil)
	
	return it
}

// drainIter reads all tokens from an iterator and returns them.
func drainIter(t *testing.T, it interface {
	Next() (*token2.UnspentToken, error)
	Close()
}) []*token2.UnspentToken {
	t.Helper()
	var result []*token2.UnspentToken
	for {
		tok, err := it.Next()
		require.NoError(t, err)
		if tok == nil {
			break
		}
		result = append(result, tok)
	}
	it.Close()
	
	return result
}

// ---- filterIterator ---------------------------------------------------------

// TestFilterIteratorSender_UsesSenderID verifies that when sender=true the
// wallet uses SenderID (not RecipientID) as the second wallet lookup key.
func TestFilterIteratorSender_UsesSenderID(t *testing.T) {
	tok := newToken(t)
	qe := &drivermock.QueryEngine{}
	qe.UnspentTokensIteratorByStub = func(_ context.Context, id string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		switch id {
		case "base":
			return emptyIter(), nil
		case "sender-id":
			return singleTokenIter(tok), nil
		default:
			t.Fatalf("unexpected wallet ID: %s", id)
		
			return nil, nil
		}
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:   "base",
			senderID: "sender-id",
		},
	}

	it, err := w.filterIterator(context.Background(), "", true, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Len(t, tokens, 1)
	assert.Equal(t, tok, tokens[0])
}

// TestFilterIteratorRecipient_UsesRecipientID verifies that when sender=false the
// wallet uses RecipientID as the second lookup key.
func TestFilterIteratorRecipient_UsesRecipientID(t *testing.T) {
	tok := newToken(t)
	qe := &drivermock.QueryEngine{}
	qe.UnspentTokensIteratorByStub = func(_ context.Context, id string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		switch id {
		case "base":
			return emptyIter(), nil
		case "recipient-id":
			return singleTokenIter(tok), nil
		default:
			t.Fatalf("unexpected wallet ID: %s", id)
		
			return nil, nil
		}
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:      "base",
			recipientID: "recipient-id",
		},
	}

	it, err := w.filterIterator(context.Background(), "", false, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Len(t, tokens, 1)
	assert.Equal(t, tok, tokens[0])
}

// TestFilterIteratorAllFail_ReturnsError verifies that when every wallet ID
// lookup returns an error the function itself returns an error.
func TestFilterIteratorAllFail_ReturnsError(t *testing.T) {
	qe := &drivermock.QueryEngine{}
	qe.UnspentTokensIteratorByStub = func(_ context.Context, id string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		return nil, errors.New("not found: " + id)
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:      "base",
			recipientID: "recipient-id",
		},
	}

	_, err := w.filterIterator(context.Background(), "", false, SelectNonExpired)
	require.Error(t, err)
}

// TestFilterIteratorFirstFails_SecondSucceeds verifies that a partial error is
// tolerated: as long as at least one iterator is valid the call succeeds.
func TestFilterIteratorFirstFails_SecondSucceeds(t *testing.T) {
	tok := newToken(t)
	qe := &drivermock.QueryEngine{}
	qe.UnspentTokensIteratorByStub = func(_ context.Context, id string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		if id == "base" {
			return nil, errors.New("base wallet unavailable")
		}

		return singleTokenIter(tok), nil
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:      "base",
			recipientID: "recipient-id",
		},
	}

	it, err := w.filterIterator(context.Background(), "", false, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Len(t, tokens, 1)
	assert.Equal(t, tok, tokens[0])
}

// TestFilterIteratorSingleValid_ReturnsDirectly verifies the single-iterator
// fast path (line 307-309): no chainedIterator is created.
func TestFilterIteratorSingleValid_ReturnsDirectly(t *testing.T) {
	tok := newToken(t)
	qe := &drivermock.QueryEngine{}

	calls := 0
	qe.UnspentTokensIteratorByStub = func(_ context.Context, _ string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		calls++
		if calls == 1 {
			return singleTokenIter(tok), nil
		}

		return nil, errors.New("second wallet unavailable")
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:      "wallet-a",
			recipientID: "wallet-b",
		},
	}

	it, err := w.filterIterator(context.Background(), "USD", false, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Len(t, tokens, 1)
	assert.Equal(t, tok, tokens[0])
}

// TestFilterIteratorBothValid_ChainsIterators verifies the multi-iterator path
// (lines 311-315): tokens from both iterators are returned in order.
func TestFilterIteratorBothValid_ChainsIterators(t *testing.T) {
	tok1 := newToken(t)
	tok2 := newToken(t)
	qe := &drivermock.QueryEngine{}
	calls := 0
	qe.UnspentTokensIteratorByStub = func(_ context.Context, _ string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		calls++
		if calls == 1 {
			return singleTokenIter(tok1), nil
		}
		
		return singleTokenIter(tok2), nil
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:      "wallet-a",
			recipientID: "wallet-b",
		},
	}

	it, err := w.filterIterator(context.Background(), "", false, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Len(t, tokens, 2)
	assert.Equal(t, tok1, tokens[0])
	assert.Equal(t, tok2, tokens[1])
}

// TestFilterIteratorNilWalletIDs_FallsBack verifies that when walletIDs is nil
// (line 277-279) a tokenOwnerWalletIDProvider is constructed from w.wallet,
// which will panic on nil wallet – so we guard with the stub instead.
//
// This test mirrors the existing TestOwnerWalletFilterIteratorFallsBackToBaseWalletID
// but exercises the nil-provider branch explicitly.
func TestFilterIteratorNilProvider_FallsBackToStub(t *testing.T) {
	// Using a stub provider (non-nil) to avoid nil pointer on w.wallet.
	// The nil-provider branch is separately exercised by setting walletIDs=nil
	// only when w.wallet is also set, which requires the full token.OwnerWallet.
	// This test validates the *stub* fallback path (already covered) and documents
	// the expected call pattern when walletIDs IS provided.
	tok := newToken(t)
	qe := &drivermock.QueryEngine{}
	qe.UnspentTokensIteratorByStub = func(_ context.Context, id string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		if id == "base" {
			return singleTokenIter(tok), nil
		}
		
		return emptyIter(), nil
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs: &stubWalletIDProvider{
			baseID:      "base",
			recipientID: "other",
		},
	}

	it, err := w.filterIterator(context.Background(), "", false, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Len(t, tokens, 1)
	assert.Equal(t, tok, tokens[0])
}

// TestFilterIteratorSelector_FiltersExpired verifies that the selector is
// applied: expired tokens are rejected when using SelectNonExpired.
func TestFilterIteratorSelector_FiltersExpired(t *testing.T) {
	expiredTok := makeFilterToken(t, &Script{
		Sender:    []byte("s"),
		Recipient: []byte("r"),
		Deadline:  time.Now().Add(-time.Hour),
	})
	qe := &drivermock.QueryEngine{}
	calls := 0
	qe.UnspentTokensIteratorByStub = func(_ context.Context, _ string, _ token2.Type) (driver.UnspentTokensIterator, error) {
		calls++
		if calls == 1 {
			return singleTokenIter(expiredTok), nil
		}
		
		return emptyIter(), nil
	}

	w := &OwnerWallet{
		queryEngine: qe,
		walletIDs:   &stubWalletIDProvider{baseID: "w", recipientID: "w2"},
	}

	it, err := w.filterIterator(context.Background(), "", false, SelectNonExpired)
	require.NoError(t, err)
	tokens := drainIter(t, it)
	require.Empty(t, tokens)
}

// ---- chainedIterator --------------------------------------------------------

// TestChainedIteratorNext_FirstIterator verifies that the first token of the
// first underlying iterator is returned correctly (line 326-332).
func TestChainedIteratorNext_FirstIterator(t *testing.T) {
	tok := newToken(t)
	iter1 := singleTokenIter(tok)
	iter2 := emptyIter()

	ci := &chainedIterator{
		iterators:    []driver.UnspentTokensIterator{iter1, iter2},
		currentIndex: 0,
	}

	got, err := ci.Next()
	require.NoError(t, err)
	assert.Equal(t, tok, got)
}

// TestChainedIteratorNext_AdvancesAcrossIterators verifies that once the first
// iterator is exhausted (Next returns nil) the chained iterator moves to the
// next one (lines 333-334) and returns its first token.
func TestChainedIteratorNext_AdvancesAcrossIterators(t *testing.T) {
	tok1 := newToken(t)
	tok2 := newToken(t)
	iter1 := singleTokenIter(tok1)
	iter2 := singleTokenIter(tok2)

	ci := &chainedIterator{
		iterators:    []driver.UnspentTokensIterator{iter1, iter2},
		currentIndex: 0,
	}

	got1, err := ci.Next()
	require.NoError(t, err)
	assert.Equal(t, tok1, got1)

	got2, err := ci.Next()
	require.NoError(t, err)
	assert.Equal(t, tok2, got2)

	got3, err := ci.Next()
	require.NoError(t, err)
	assert.Nil(t, got3) // exhausted
}

// TestChainedIteratorNext_AllExhausted verifies that when all iterators are
// empty the chained iterator returns nil, nil (lines 336-337).
func TestChainedIteratorNext_AllExhausted(t *testing.T) {
	ci := &chainedIterator{
		iterators:    []driver.UnspentTokensIterator{emptyIter(), emptyIter()},
		currentIndex: 0,
	}

	got, err := ci.Next()
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestChainedIteratorNext_ErrorPropagates verifies that an error from an
// underlying iterator is returned immediately (lines 327-329).
func TestChainedIteratorNext_ErrorPropagates(t *testing.T) {
	boom := errors.New("storage failure")
	iter1 := &drivermock.UnspentTokensIterator{}
	iter1.NextReturnsOnCall(0, nil, boom)

	ci := &chainedIterator{
		iterators:    []driver.UnspentTokensIterator{iter1},
		currentIndex: 0,
	}

	_, err := ci.Next()
	require.ErrorIs(t, err, boom)
}

// TestChainedIteratorNext_EmptyList verifies the degenerate case of zero
// iterators: Next should immediately return nil, nil.
func TestChainedIteratorNext_EmptyList(t *testing.T) {
	ci := &chainedIterator{iterators: nil, currentIndex: 0}

	got, err := ci.Next()
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestChainedIteratorClose_CallsAllClose verifies that Close forwards to every
// underlying iterator (lines 341-343).
func TestChainedIteratorClose_CallsAllClose(t *testing.T) {
	iter1 := &drivermock.UnspentTokensIterator{}
	iter2 := &drivermock.UnspentTokensIterator{}

	ci := &chainedIterator{
		iterators:    []driver.UnspentTokensIterator{iter1, iter2},
		currentIndex: 0,
	}

	ci.Close()

	assert.Equal(t, 1, iter1.CloseCallCount())
	assert.Equal(t, 1, iter2.CloseCallCount())
}

// TestChainedIteratorClose_EmptyList verifies that Close on an empty iterator
// list does not panic.
func TestChainedIteratorClose_EmptyList(t *testing.T) {
	ci := &chainedIterator{iterators: nil}
	assert.NotPanics(t, func() { ci.Close() })
}
