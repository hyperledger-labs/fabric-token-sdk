/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger("token-sdk.sql")

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
	nc := db.NewTableNameCreator()

	return tableNames{
		Movements:              nc.MustGetTableName(prefix, "movements"),
		Transactions:           nc.MustGetTableName(prefix, "transactions"),
		TransactionEndorseAck:  nc.MustGetTableName(prefix, "transaction_endorsements"),
		Requests:               nc.MustGetTableName(prefix, "requests"),
		Validations:            nc.MustGetTableName(prefix, "request_validations"),
		Tokens:                 nc.MustGetTableName(prefix, "tokens"),
		Ownership:              nc.MustGetTableName(prefix, "token_ownership"),
		Certifications:         nc.MustGetTableName(prefix, "token_certifications"),
		TokenLocks:             nc.MustGetTableName(prefix, "token_locks"),
		PublicParams:           nc.MustGetTableName(prefix, "public_params"),
		Wallets:                nc.MustGetTableName(prefix, "wallets"),
		IdentityConfigurations: nc.MustGetTableName(prefix, "identity_configurations"),
		IdentityInfo:           nc.MustGetTableName(prefix, "identity_information"),
		Signers:                nc.MustGetTableName(prefix, "identity_signers"),
	}, nil
}
