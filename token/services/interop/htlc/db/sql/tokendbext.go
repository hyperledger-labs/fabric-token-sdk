/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/ext"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.sql.htlc")

type tableNames struct {
	Tokens string
}

type HTLCTokenDBExtensionFactory struct{}

func NewHTLCTokenDBExtensionFactory() *HTLCTokenDBExtensionFactory {
	return &HTLCTokenDBExtensionFactory{}
}

func (H *HTLCTokenDBExtensionFactory) NewExtension(prefix string) (ext.TokenDBExtension, error) {
	return NewHTLCTokenDBExtension(prefix)
}

type HTLCTokenDBExtension struct {
	table tableNames
}

func NewHTLCTokenDBExtension(prefix string) (*HTLCTokenDBExtension, error) {
	table, err := getTableNames(prefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting table names for prefix [%s]", prefix)
	}
	return &HTLCTokenDBExtension{table: table}, nil
}

func (e *HTLCTokenDBExtension) GetSchema(string) string {
	return fmt.Sprintf(`
		-- Tokens
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			sender_raw BYTEA NOT NULL,
			recipient_raw BYTEA NOT NULL,
			deadline TIMESTAMP NOT NULL,
			hash BYTEA NOT NULL,
			hash_function INT NOT NULL,
			hash_encoding INT NOT NULL,
			PRIMARY KEY (tx_id, idx)
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );
		`,
		e.table.Tokens,
		e.table.Tokens, e.table.Tokens,
	)
}

func (e *HTLCTokenDBExtension) Delete(tx *sql.Tx, txID string, index uint64, deletedBy string) error {
	return nil
}

func (e *HTLCTokenDBExtension) StoreToken(tx *sql.Tx, tr driver.TokenRecord, owners []string) error {
	if tr.OwnerType != htlc.ScriptType {
		// nothing to store here
		return nil
	}
	script := &htlc.Script{}
	if err := json.Unmarshal(tr.OwnerIdentity, script); err != nil {
		return errors.Wrapf(err, "failed to unmrshal HTLC script")
	}
	// store the script
	logger.Debugf("store htlc record [%s:%d,%v] in table [%s]", tr.TxID, tr.Index, owners, e.table.Tokens)

	// Store token
	query := fmt.Sprintf("INSERT INTO %s (tx_id, idx, sender_raw, recipient_raw, deadline, hash, hash_function, hash_encoding) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", e.table.Tokens)
	logger.Debug(query,
		tr.TxID,
		tr.Index,
		script.Sender,
		script.Recipient,
		script.Deadline,
		script.HashInfo.Hash,
		script.HashInfo.HashFunc,
		script.HashInfo.HashEncoding)
	if _, err := tx.Exec(query,
		tr.TxID,
		tr.Index,
		script.Sender,
		script.Recipient,
		script.Deadline,
		script.HashInfo.Hash,
		script.HashInfo.HashFunc,
		script.HashInfo.HashEncoding); err != nil {
		logger.Errorf("error storing htlc token [%s] in table [%s]: [%s][%s]", tr.TxID, e.table.Tokens, err, string(debug.Stack()))
		return errors.Wrapf(err, "error storing htlc token [%s] in table [%s]", tr.TxID, e.table.Tokens)
	}

	return nil
}

func (e *HTLCTokenDBExtension) DeleteTokens(tx *sql.Tx, deletedBy string, ids ...*token.ID) error {
	//TODO implement me
	panic("implement me")
}

func getTableNames(prefix string) (tableNames, error) {
	if prefix != "" {
		if len(prefix) > 100 {
			return tableNames{}, errors.New("table prefix must be shorter than 100 characters")
		}
		r := regexp.MustCompile("^[a-zA-Z_]+$")
		if !r.MatchString(prefix) {
			return tableNames{}, errors.New("illegal character in table prefix, only letters and underscores allowed")
		}
		prefix = strings.ToLower(prefix) + "_"
	}

	return tableNames{
		Tokens: fmt.Sprintf("%stokens", prefix),
	}, nil
}
