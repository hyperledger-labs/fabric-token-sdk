/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.sql")

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
	for _, schema := range schemas {
		logger.Debug(schema)
		if _, err = tx.Exec(schema); err != nil {
			return errors.Wrap(err, "error creating schema")
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
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
	IdentityInfo           string
	Signers                string
	TokenLocks             string
}

func GetTableNames(prefix string) (tableNames, error) {
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
		Movements:              fmt.Sprintf("%smovements", prefix),
		Transactions:           fmt.Sprintf("%stransactions", prefix),
		TransactionEndorseAck:  fmt.Sprintf("%stransaction_endorsements", prefix),
		Requests:               fmt.Sprintf("%srequests", prefix),
		Validations:            fmt.Sprintf("%srequest_validations", prefix),
		Tokens:                 fmt.Sprintf("%stokens", prefix),
		Ownership:              fmt.Sprintf("%stoken_ownership", prefix),
		Certifications:         fmt.Sprintf("%stoken_certifications", prefix),
		TokenLocks:             fmt.Sprintf("%stoken_locks", prefix),
		PublicParams:           fmt.Sprintf("%spublic_params", prefix),
		Wallets:                fmt.Sprintf("%swallets", prefix),
		IdentityConfigurations: fmt.Sprintf("%sidentity_configurations", prefix),
		IdentityInfo:           fmt.Sprintf("%sidentity_information", prefix),
		Signers:                fmt.Sprintf("%sidentity_signers", prefix),
	}, nil
}
