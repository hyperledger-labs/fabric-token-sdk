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
		Transactions:           nc.MustGetTableName(prefix, "txs", params...),
		TransactionEndorseAck:  nc.MustGetTableName(prefix, "tx_ends", params...),
		Requests:               nc.MustGetTableName(prefix, "requests", params...),
		Validations:            nc.MustGetTableName(prefix, "req_vals", params...),
		Tokens:                 nc.MustGetTableName(prefix, "tokens", params...),
		Ownership:              nc.MustGetTableName(prefix, "tkn_own", params...),
		Certifications:         nc.MustGetTableName(prefix, "tkn_crts", params...),
		TokenLocks:             nc.MustGetTableName(prefix, "tkn_locks", params...),
		PublicParams:           nc.MustGetTableName(prefix, "public_params", params...),
		Wallets:                nc.MustGetTableName(prefix, "wallets", params...),
		IdentityConfigurations: nc.MustGetTableName(prefix, "id_cfgs", params...),
		IdentityInfo:           nc.MustGetTableName(prefix, "id_info", params...),
		Signers:                nc.MustGetTableName(prefix, "id_signers", params...),
	}, nil
}
