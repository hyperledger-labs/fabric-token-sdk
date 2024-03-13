/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

	fmt.Println("schema")
	for _, schema := range schemas {
		logger.Debug(schema)
		fmt.Println(schema)
		if _, err = db.Exec(schema); err != nil {
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
	AuditInfo              string
	Signers                string
}

func getTableNames(prefix, name string) (tableNames, error) {
	if prefix != "" {
		r := regexp.MustCompile("^[a-zA-Z_]+$")
		if !r.MatchString(prefix) {
			return tableNames{}, errors.New("illegal character in table prefix, only letters and underscores allowed")
		}
		prefix = prefix + "_"
	}

	// name is usually something like "default,testchannel,token-chaincode",
	// so we shorten it to the first 6 hex characters of the hash.
	h := sha256.New()
	if _, err := h.Write([]byte(name)); err != nil {
		return tableNames{}, errors.Wrapf(err, "error hashing name [%s]", name)
	}
	suffix := "_" + hex.EncodeToString(h.Sum(nil)[:3])

	return tableNames{
		Movements:              fmt.Sprintf("%smovements%s", prefix, suffix),
		Transactions:           fmt.Sprintf("%stransactions%s", prefix, suffix),
		Requests:               fmt.Sprintf("%srequests%s", prefix, suffix),
		Validations:            fmt.Sprintf("%svalidations%s", prefix, suffix),
		TransactionEndorseAck:  fmt.Sprintf("%stea%s", prefix, suffix),
		Certifications:         fmt.Sprintf("%scertifications%s", prefix, suffix),
		Tokens:                 fmt.Sprintf("%stokens%s", prefix, suffix),
		Ownership:              fmt.Sprintf("%sownership%s", prefix, suffix),
		PublicParams:           fmt.Sprintf("%spublic_params%s", prefix, suffix),
		Wallets:                fmt.Sprintf("%swallet%s", prefix, suffix),
		IdentityConfigurations: fmt.Sprintf("%s%sd_configs%s", prefix, "i", suffix),
		AuditInfo:              fmt.Sprintf("%saudit_info%s", prefix, suffix),
		Signers:                fmt.Sprintf("%ssigners%s", prefix, suffix),
	}, nil
}
