/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	driver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	sqlcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
	tokentype "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	_ "modernc.org/sqlite"
)

func openBenchTokenStore(b *testing.B) (*sqlcommon.TokenStore, func()) {
	b.Helper()
	dir := b.TempDir()
	dsn := fmt.Sprintf("file:%s/bench.sqlite?_pragma=busy_timeout(20000)", dir)
	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		b.Fatal(err)
	}
	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		b.Fatal(err)
	}
	tables, err := sqlcommon.GetTableNames("")
	if err != nil {
		b.Fatal(err)
	}
	store, err := sqlcommon.NewTokenStoreWithNotifier(readDB, writeDB, tables, NewConditionInterpreter(), nil)
	if err != nil {
		b.Fatal(err)
	}
	if err := store.CreateSchema(); err != nil {
		b.Fatal(err)
	}

	return store, func() {
		_ = readDB.Close()
		_ = writeDB.Close()
	}
}

func seedBenchTokens(b *testing.B, store *sqlcommon.TokenStore, n int) {
	b.Helper()
	ctx := context.Background()
	for i := range n {
		rec := driver.TokenRecord{
			TxID:           fmt.Sprintf("tx%d", i),
			Index:          0,
			OwnerRaw:       []byte("wallet0"),
			OwnerType:      "idemix",
			OwnerIdentity:  []byte("wallet0"),
			OwnerWalletID:  "wallet0",
			Ledger:         []byte("ledger"),
			LedgerMetadata: []byte("meta"),
			Quantity:       "0x64",
			Type:           tokentype.Type("GOLD"),
		}
		if err := store.StoreToken(ctx, rec, []string{"wallet0"}); err != nil {
			b.Fatalf("seed failed at i=%d: %v", i, err)
		}
	}
}

func BenchmarkUnspentTokensIterator(b *testing.B) {
	store, cleanup := openBenchTokenStore(b)
	defer cleanup()
	seedBenchTokens(b, store, 1000)

	cfg := benchmark.NewConfig(4, 5*time.Second, 500*time.Millisecond)
	result := benchmark.RunBenchmark(cfg,
		func() *sqlcommon.TokenStore { return store },
		func(s *sqlcommon.TokenStore) error {
			it, err := s.UnspentTokensIterator(context.Background())
			if err != nil {
				return err
			}
			defer it.Close()
			for {
				tok, err := it.Next()
				if err != nil {
					return err
				}
				if tok == nil {
					break
				}
			}

			return nil
		},
	)
	result.Print()
}

func BenchmarkBalance(b *testing.B) {
	store, cleanup := openBenchTokenStore(b)
	defer cleanup()
	seedBenchTokens(b, store, 1000)

	cfg := benchmark.NewConfig(4, 5*time.Second, 500*time.Millisecond)
	result := benchmark.RunBenchmark(cfg,
		func() *sqlcommon.TokenStore { return store },
		func(s *sqlcommon.TokenStore) error {
			_, err := s.Balance(context.Background(), "wallet0", tokentype.Type("GOLD"))

			return err
		},
	)
	result.Print()
}

func BenchmarkUnspentTokensIteratorBy(b *testing.B) {
	store, cleanup := openBenchTokenStore(b)
	defer cleanup()
	seedBenchTokens(b, store, 1000)

	cfg := benchmark.NewConfig(4, 5*time.Second, 500*time.Millisecond)
	result := benchmark.RunBenchmark(cfg,
		func() *sqlcommon.TokenStore { return store },
		func(s *sqlcommon.TokenStore) error {
			it, err := s.UnspentTokensIteratorBy(context.Background(), "wallet0", tokentype.Type("GOLD"))
			if err != nil {
				return err
			}
			defer it.Close()
			for {
				tok, err := it.Next()
				if err != nil {
					return err
				}
				if tok == nil {
					break
				}
			}

			return nil
		},
	)
	result.Print()
}
