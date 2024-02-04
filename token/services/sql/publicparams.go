package sql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type ParamsDB struct {
	table string
	db    *sql.DB
}

func newPublicParamsDB(db *sql.DB, table string) *ParamsDB {
	return &ParamsDB{
		table: table,
		db:    db,
	}
}

func (db *ParamsDB) StorePublicParams(raw []byte) error {
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (raw, stored_at) VALUES ($1, $2)", db.table)
	logger.Debug(query, fmt.Sprintf("(%d bytes), %v", len(raw), now))

	_, err := db.db.Exec(query, raw, now)
	return err
}

func (db *ParamsDB) GetRawPublicParams() ([]byte, error) {
	var params []byte
	query := fmt.Sprintf("SELECT raw FROM %s ORDER BY stored_at DESC LIMIT 1;", db.table)
	logger.Debug(query)

	row := db.db.QueryRow(query)
	err := row.Scan(&params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Errorf("params not found")
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return params, nil
}

func (db *ParamsDB) GetSchema() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			raw BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (raw)
		);
		`,
		db.table,
	)
}
