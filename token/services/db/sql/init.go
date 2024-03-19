/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"regexp"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.sql")

func initSchema(db *sql.DB, schemas ...string) (err error) {
	logger.Info("creating tables")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil && tx != nil {
			if err := tx.Rollback(); err != nil {
				logger.Errorf("failed to rollback [%s][%s]", err, debug.Stack())
			}
		}
	}()
	for i, schema := range schemas {
		logger.Debugf("schema %d: %s", i, schema)
		if _, err = db.Exec(schema); err != nil {
			return errors.Wrap(err, "error creating schema")
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	logger.Debug("tables created")
	return
}

type tableNames struct {
	Movements              string
	Transactions           string
	Requests               string
	Validations            string
	TransactionEndorseAck  string
	Certifications         string
	Tokens                 string
	Ownership              string
	PublicParams           string
	Wallets                string
	IdentityConfigurations string
	AuditInfo              string
	Signers                string
}

func getTableNames(prefix string) (tableNames, error) {
	if prefix != "" {
		r := regexp.MustCompile("^[a-zA-Z_]+$")
		if !r.MatchString(prefix) {
			return tableNames{}, errors.New("illegal character in table prefix, only letters and underscores allowed")
		}
		prefix = prefix + "_"
	}

	return tableNames{
		Movements:              fmt.Sprintf("%smovements", prefix),
		Transactions:           fmt.Sprintf("%stransactions", prefix),
		Requests:               fmt.Sprintf("%srequests", prefix),
		Validations:            fmt.Sprintf("%svalidations", prefix),
		TransactionEndorseAck:  fmt.Sprintf("%stea", prefix),
		Certifications:         fmt.Sprintf("%scertifications", prefix),
		Tokens:                 fmt.Sprintf("%stokens", prefix),
		Ownership:              fmt.Sprintf("%sownership", prefix),
		PublicParams:           fmt.Sprintf("%spublic_params", prefix),
		Wallets:                fmt.Sprintf("%swallet", prefix),
		IdentityConfigurations: fmt.Sprintf("%sid_configs", prefix),
		AuditInfo:              fmt.Sprintf("%saudit_info", prefix),
		Signers:                fmt.Sprintf("%ssigners", prefix),
	}, nil
}
