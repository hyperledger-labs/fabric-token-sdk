/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"
	"strconv"
	"sync/atomic"

	scommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	tokensdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	sqlcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
	token "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TokenStore wraps common.TokenStore to add advisory lock to schema creation
type TokenStore struct {
	*sqlcommon.TokenStore
	writeDB *sql.DB
	lockID  int64
}

// GetSchema overrides the base GetSchema to prefix with advisory lock
func (s *TokenStore) GetSchema() string {
	baseSchema := s.TokenStore.GetSchema()

	return prefixSchemaWithLock(baseSchema, s.lockID)
}

// CreateSchema overrides the base CreateSchema to ensure GetSchema is called on the correct receiver
func (s *TokenStore) CreateSchema() error {
	return common.InitSchema(s.writeDB, s.GetSchema())
}

// TokenNotifier handles notifications for tokens.
type TokenNotifier struct {
	*Notifier
}

// NewTokenNotifier returns a new TokenNotifier for the given RWDB and table names.
// It includes owner_wallet_id, token_type, and quantity in the NOTIFY payload so
// subscribers can perform surgical cache updates without an extra DB query.
func NewTokenNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, dataSource string) (*TokenNotifier, error) {
	return &TokenNotifier{
		Notifier: NewNotifier(
			dbs.WriteDB,
			tableNames.Tokens,
			dataSource,
			AllOperations,
			*NewSimplePrimaryKey("tx_id"),
			*NewSimplePrimaryKey("idx"),
			*NewSimplePrimaryKey("owner_wallet_id"),
			*NewSimplePrimaryKey("token_type"),
			*NewSimplePrimaryKey("quantity"),
		),
	}, nil
}

// Subscribe registers a callback and returns a cancel function that silences only
// this subscription. The underlying Postgres LISTEN goroutine keeps running until
// the TokenNotifier itself is closed; calling cancel merely stops dispatching to
// this particular callback, so other subscribers are not affected.
func (n *TokenNotifier) Subscribe(callback func(tokensdriver.Operation, tokensdriver.TokenRecordReference)) (func() error, error) {
	var active atomic.Bool
	active.Store(true)

	err := n.Notifier.Subscribe(func(operation tokensdriver.Operation, m map[tokensdriver.ColumnKey]string) {
		if !active.Load() {
			return
		}
		idx, err := strconv.ParseUint(m["idx"], 10, 64)
		if err != nil {
			logger.Errorf("failed to parse token index [%s]: %s", m["idx"], err)

			return
		}
		callback(operation, tokensdriver.TokenRecordReference{
			TxID:     m["tx_id"],
			Index:    idx,
			WalletID: m["owner_wallet_id"],
			Type:     token.Type(m["token_type"]),
			Quantity: m["quantity"],
		})
	})
	if err != nil {
		return nil, err
	}

	return func() error {
		active.Store(false)

		return nil
	}, nil
}

func NewTokenStoreWithNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, notifier *TokenNotifier) (*TokenStore, error) {
	baseStore, err := sqlcommon.NewTokenStoreWithNotifier(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		notifier,
	)
	if err != nil {
		return nil, err
	}

	// Wrap with postgres-specific store that adds advisory lock to schema
	return &TokenStore{
		TokenStore: baseStore,
		writeDB:    dbs.WriteDB,
		lockID:     createTableLockID("tokens"),
	}, nil
}
