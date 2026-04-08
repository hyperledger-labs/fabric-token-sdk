/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokenlockdb"
	tx "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type unitTestMockTokenDB struct {
	SpendableTokensIteratorByStub func(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error)
}

func (m *unitTestMockTokenDB) SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
	if m.SpendableTokensIteratorByStub != nil {
		return m.SpendableTokensIteratorByStub(ctx, walletID, typ)
	}
	return nil, nil
}

type unitTestMockSpendableTokensIterator struct {
	driver.SpendableTokensIterator
	NextStub func() (*token2.UnspentTokenInWallet, error)
}

func (m *unitTestMockSpendableTokensIterator) Next() (*token2.UnspentTokenInWallet, error) {
	return m.NextStub()
}
func (m *unitTestMockSpendableTokensIterator) Close() {}

type unitTestMockMetricsProvider struct {
	metrics.Provider
}

func (m *unitTestMockMetricsProvider) NewCounter(opts metrics.CounterOpts) metrics.Counter {
	return &unitTestMockCounter{}
}
func (m *unitTestMockMetricsProvider) NewHistogram(opts metrics.HistogramOpts) metrics.Histogram {
	return &unitTestMockHistogram{}
}

type unitTestMockCounter struct {
	metrics.Counter
}

func (c *unitTestMockCounter) With(labelValues ...string) metrics.Counter { return c }
func (c *unitTestMockCounter) Add(delta float64)                          {}

type unitTestMockHistogram struct {
	metrics.Histogram
}

func (h *unitTestMockHistogram) With(labelValues ...string) metrics.Histogram { return h }
func (h *unitTestMockHistogram) Observe(value float64)                        {}

func TestLazyFetcherUnit(t *testing.T) {
	mockDB := &unitTestMockTokenDB{}
	fetcher := NewLazyFetcher(mockDB)

	t.Run("FetchSuccess", func(t *testing.T) {
		mockIt := &unitTestMockSpendableTokensIterator{}
		var count int
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			count++
			if count > 1 {
				return nil, nil
			}
			return &token2.UnspentTokenInWallet{
				Id:       token2.ID{TxId: "tx1", Index: 0},
				Type:     "ABC",
				Quantity: "100",
			}, nil
		}
		mockDB.SpendableTokensIteratorByStub = func(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
			return mockIt, nil
		}

		it, err := fetcher.UnspentTokensIteratorBy(context.Background(), "alice", "ABC")
		require.NoError(t, err)
		tok, err := it.Next()
		require.NoError(t, err)
		assert.Equal(t, "tx1", tok.Id.TxId)
	})

	t.Run("FetchError", func(t *testing.T) {
		mockDB.SpendableTokensIteratorByStub = func(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
			return nil, errors.New("db error")
		}
		_, err := fetcher.UnspentTokensIteratorBy(context.Background(), "alice", "ABC")
		require.Error(t, err)
	})
}

func TestCachedFetcherUnit(t *testing.T) {
	mockDB := &unitTestMockTokenDB{}
	fetcher := newCachedFetcher(mockDB, 10, 100*time.Millisecond, 5)

	t.Run("FetchSuccess", func(t *testing.T) {
		mockIt := &unitTestMockSpendableTokensIterator{}
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			return nil, nil
		}
		mockDB.SpendableTokensIteratorByStub = func(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
			return mockIt, nil
		}

		_, err := fetcher.UnspentTokensIteratorBy(context.Background(), "alice", "ABC")
		require.NoError(t, err)
	})
}

func TestMixedFetcherUnit(t *testing.T) {
	mockDB := &unitTestMockTokenDB{}
	metrics := NewMetrics(&unitTestMockMetricsProvider{})
	fetcher := newMixedFetcher(mockDB, metrics, 10, 100*time.Millisecond, 5)

	t.Run("FetchSuccess", func(t *testing.T) {
		mockIt := &unitTestMockSpendableTokensIterator{}
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			return nil, nil
		}
		mockDB.SpendableTokensIteratorByStub = func(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
			return mockIt, nil
		}

		_, err := fetcher.UnspentTokensIteratorBy(context.Background(), "alice", "ABC")
		require.NoError(t, err)
	})
}

func TestSelectorUnit(t *testing.T) {
	mockFetcher := &unitTestMockFetcher{}
	mockLocker := &unitTestMockLocker{}
	metrics := NewMetrics(&unitTestMockMetricsProvider{})
	s := NewSelector(logger, mockFetcher, mockLocker, 64, metrics)

	t.Run("SelectSuccess", func(t *testing.T) {
		mockIt := &unitTestMockIterator{}
		var count int
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			count++
			if count > 1 {
				return nil, nil
			}
			return &token2.UnspentTokenInWallet{
				Id:       token2.ID{TxId: "tx1", Index: 0},
				Type:     "ABC",
				Quantity: "100",
			}, nil
		}
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			return mockIt, nil
		}
		mockLocker.TryLockStub = func(ctx context.Context, id *token2.ID) bool { return true }

		tokens, sum, err := s.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, "100", sum.Decimal())
	})

	t.Run("InsufficientFunds", func(t *testing.T) {
		mockIt := &unitTestMockIterator{}
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) { return nil, nil }
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			return mockIt, nil
		}
		_, _, err := s.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient funds")
	})

	t.Run("ClosedError", func(t *testing.T) {
		s2 := NewSelector(logger, mockFetcher, mockLocker, 2, metrics)
		s2.Close()
		_, _, err := s2.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "selector is already closed")
	})

	t.Run("FetcherError", func(t *testing.T) {
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			return nil, errors.New("fetcher error")
		}
		_, _, err := s.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetcher error")
	})

	t.Run("CacheNextError", func(t *testing.T) {
		mockIt := &unitTestMockIterator{}
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			return nil, errors.New("next error")
		}
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			return mockIt, nil
		}

		s3 := NewSelector(logger, mockFetcher, mockLocker, 64, metrics)
		_, _, err := s3.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "next error")
	})
}

func TestStubbornSelectorUnit(t *testing.T) {
	mockFetcher := &unitTestMockFetcher{}
	mockLocker := &unitTestMockLocker{}
	metrics := NewMetrics(&unitTestMockMetricsProvider{})
	s := NewStubbornSelector(logger, mockFetcher, mockLocker, 64, 100*time.Millisecond, 2, metrics)

	t.Run("SelectSuccessAfterImmediateRetries", func(t *testing.T) {
		mockIt := &unitTestMockIterator{}
		var countNext int
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			countNext++
			if countNext > 1 {
				return nil, nil
			}
			return &token2.UnspentTokenInWallet{
				Id:       token2.ID{TxId: "tx1", Index: 0},
				Type:     "ABC",
				Quantity: "100",
			}, nil
		}
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			// Reset countNext for each retry
			countNext = 0
			return mockIt, nil
		}
		// Fails first time
		var countLock int
		mockLocker.TryLockStub = func(ctx context.Context, id *token2.ID) bool {
			countLock++
			return countLock > 1
		}

		tokens, sum, err := s.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, "100", sum.Decimal())
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockIt := &unitTestMockIterator{}
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			return nil, ctx.Err()
		}
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			return mockIt, nil
		}

		_, _, err := s.Select(ctx, &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.Error(t, err)
	})

	t.Run("MaxRetriesExceeded", func(t *testing.T) {
		mockIt := &unitTestMockIterator{}
		var countNext int
		mockIt.NextStub = func() (*token2.UnspentTokenInWallet, error) {
			countNext++
			if countNext > 1 {
				return nil, nil
			}
			return &token2.UnspentTokenInWallet{
				Id:       token2.ID{TxId: "tx1", Index: 0},
				Type:     "ABC",
				Quantity: "100",
			}, nil
		}
		mockFetcher.UnspentTokensIteratorByStub = func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			countNext = 0
			return mockIt, nil
		}
		mockLocker.TryLockStub = func(ctx context.Context, id *token2.ID) bool { return false }

		// To trigger SelectorSufficientButLockedFunds, we need to exceed maxImmediateRetries (5)
		// But selectInternal returns token.SelectorSufficientButLockedFunds.
		// StubbornSelector then retries maxRetriesAfterBackoff times.
		shortBackoffS := NewStubbornSelector(logger, mockFetcher, mockLocker, 64, 1*time.Millisecond, 1, metrics)
		_, _, err := shortBackoffS.Select(context.Background(), &unitTestMockOwnerFilter{id: "alice"}, "50", "ABC")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aborted too many times")
	})
}

type unitTestMockFetcher struct {
	UnspentTokensIteratorByStub func(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error)
}

func (m *unitTestMockFetcher) UnspentTokensIteratorBy(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
	if m.UnspentTokensIteratorByStub != nil {
		return m.UnspentTokensIteratorByStub(ctx, walletID, currency)
	}
	return &unitTestMockIterator{}, nil
}

type unitTestMockLocker struct {
	TryLockStub   func(context.Context, *token2.ID) bool
	UnlockAllStub func(context.Context) error
}

func (m *unitTestMockLocker) TryLock(ctx context.Context, id *token2.ID) bool {
	if m.TryLockStub != nil {
		return m.TryLockStub(ctx, id)
	}
	return true
}
func (m *unitTestMockLocker) UnlockAll(ctx context.Context) error {
	if m.UnlockAllStub != nil {
		return m.UnlockAllStub(ctx)
	}
	return nil
}

// Locker interface for manager
func (m *unitTestMockLocker) Lock(ctx context.Context, tokenID *token2.ID, consumerTxID tx.ID) error {
	return nil
}
func (m *unitTestMockLocker) UnlockByTxID(ctx context.Context, consumerTxID tx.ID) error {
	return nil
}
func (m *unitTestMockLocker) Cleanup(ctx context.Context, leaseExpiry time.Duration) error {
	return nil
}

type unitTestMockIterator struct {
	iterator[*token2.UnspentTokenInWallet]
	NextStub func() (*token2.UnspentTokenInWallet, error)
}

func (m *unitTestMockIterator) Next() (*token2.UnspentTokenInWallet, error) { return m.NextStub() }
func (m *unitTestMockIterator) Close()                                      {}

type unitTestMockOwnerFilter struct {
	id string
}

func (f *unitTestMockOwnerFilter) ID() string { return f.id }

func TestServiceUnit(t *testing.T) {
	mockFP := &unitTestMockFetcherProvider{}
	mockLSM := &unitTestMockTokenLockStoreServiceManager{}
	mockCP := &unitTestMockConfigProvider{}
	mockM := &unitTestMockMetricsProvider{}

	svc := NewService(mockFP, mockLSM, mockCP, mockM)
	require.NotNil(t, svc)

	t.Run("Shutdown", func(t *testing.T) {
		svc.Shutdown()
		assert.Nil(t, svc.managers)
	})

	t.Run("SelectorManager_NilTMS", func(t *testing.T) {
		_, err := svc.SelectorManager(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tms")
	})

	t.Run("Loader_Load_Errors", func(t *testing.T) {
		l := &loader{
			tokenLockStoreServiceManager: mockLSM,
			fetcherProvider:              mockFP,
		}

		t.Run("PublicParametersNotSet", func(t *testing.T) {
			mockTMS := &unitTestMockTMS{}
			mockTMS.IDStub = func() token.TMSID { return token.TMSID{Network: "n1"} }
			mockTMS.PublicParametersManagerStub = func() *token.PublicParametersManager {
				return &token.PublicParametersManager{}
			}

			_, err := l.load(mockTMS)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "public parameters not set")
		})
	})
}

func TestManagerUnit(t *testing.T) {
	mockFetcher := &unitTestMockFetcher{}
	mockLocker := &unitTestMockLocker{}
	metrics := NewMetrics(&unitTestMockMetricsProvider{})

	mgr := NewManager(mockFetcher, mockLocker, 64, 0, 0, 0, 0, metrics)
	require.NotNil(t, mgr)

	t.Run("NewSelector", func(t *testing.T) {
		sel, err := mgr.NewSelector(tx.ID("tx1"))
		require.NoError(t, err)
		assert.NotNil(t, sel)
	})

	t.Run("Close_NotFound", func(t *testing.T) {
		err := mgr.Close(tx.ID("nonexistent"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Stop", func(t *testing.T) {
		mgr.Stop()
	})
}

func TestFetcherProviderUnit(t *testing.T) {
	mockSSM := &unitTestMockStoreServiceManager{}
	provider := NewFetcherProvider(mockSSM, &unitTestMockMetricsProvider{}, Mixed, 0, 0, 0)

	t.Run("GetFetcher_Error", func(t *testing.T) {
		mockSSM.StoreServiceByTMSIdStub = func(id token.TMSID) (*tokendb.StoreService, error) {
			return nil, errors.New("ssm error")
		}
		_, err := provider.GetFetcher(token.TMSID{Network: "n1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssm error")
	})
}

type unitTestMockFetcherProvider struct {
	GetFetcherStub func(tmsID token.TMSID) (tokenFetcher, error)
}

func (m *unitTestMockFetcherProvider) GetFetcher(tmsID token.TMSID) (tokenFetcher, error) {
	if m.GetFetcherStub != nil {
		return m.GetFetcherStub(tmsID)
	}
	return nil, nil
}

type unitTestMockTokenLockStoreServiceManager struct {
	StoreServiceByTMSIdStub func(id token.TMSID) (*tokenlockdb.StoreService, error)
}

func (m *unitTestMockTokenLockStoreServiceManager) StoreServiceByTMSId(id token.TMSID) (*tokenlockdb.StoreService, error) {
	if m.StoreServiceByTMSIdStub != nil {
		return m.StoreServiceByTMSIdStub(id)
	}
	return nil, nil
}

type unitTestMockTMS struct {
	IDStub                      func() token.TMSID
	PublicParametersManagerStub func() *token.PublicParametersManager
}

func (m *unitTestMockTMS) ID() token.TMSID { return m.IDStub() }
func (m *unitTestMockTMS) PublicParametersManager() *token.PublicParametersManager {
	return m.PublicParametersManagerStub()
}

type unitTestMockStoreServiceManager struct {
	tokendb.StoreServiceManager
	StoreServiceByTMSIdStub func(id token.TMSID) (*tokendb.StoreService, error)
}

func (m *unitTestMockStoreServiceManager) StoreServiceByTMSId(id token.TMSID) (*tokendb.StoreService, error) {
	return m.StoreServiceByTMSIdStub(id)
}

type unitTestMockConfigProvider struct{}

func (m *unitTestMockConfigProvider) UnmarshalKey(key string, rawVal interface{}) error { return nil }
