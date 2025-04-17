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

func GetTableNames(prefix string, params ...string) (tableNames, error) {
	nc := db.NewTableNameCreator()

	return tableNames{
		Movements:              nc.MustGetTableName(prefix, "movements", params...),
		Transactions:           nc.MustGetTableName(prefix, "transactions", params...),
		TransactionEndorseAck:  nc.MustGetTableName(prefix, "transaction_endorsements", params...),
		Requests:               nc.MustGetTableName(prefix, "requests", params...),
		Validations:            nc.MustGetTableName(prefix, "request_validations", params...),
		Tokens:                 nc.MustGetTableName(prefix, "tokens", params...),
		Ownership:              nc.MustGetTableName(prefix, "token_ownership", params...),
		Certifications:         nc.MustGetTableName(prefix, "token_certifications", params...),
		TokenLocks:             nc.MustGetTableName(prefix, "token_locks", params...),
		PublicParams:           nc.MustGetTableName(prefix, "public_params", params...),
		Wallets:                nc.MustGetTableName(prefix, "wallets", params...),
		IdentityConfigurations: nc.MustGetTableName(prefix, "identity_configurations", params...),
		IdentityInfo:           nc.MustGetTableName(prefix, "identity_information", params...),
		Signers:                nc.MustGetTableName(prefix, "identity_signers", params...),
	}, nil
}
