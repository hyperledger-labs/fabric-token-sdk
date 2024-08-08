/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
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
	nc, err := common.NewTableNameCreator(prefix)
	if err != nil {
		return tableNames{}, err
	}

	return tableNames{
		Movements:              nc.MustGetTableName("movements"),
		Transactions:           nc.MustGetTableName("transactions"),
		TransactionEndorseAck:  nc.MustGetTableName("transaction_endorsements"),
		Requests:               nc.MustGetTableName("requests"),
		Validations:            nc.MustGetTableName("request_validations"),
		Tokens:                 nc.MustGetTableName("tokens"),
		Ownership:              nc.MustGetTableName("token_ownership"),
		Certifications:         nc.MustGetTableName("token_certifications"),
		TokenLocks:             nc.MustGetTableName("token_locks"),
		PublicParams:           nc.MustGetTableName("public_params"),
		Wallets:                nc.MustGetTableName("wallets"),
		IdentityConfigurations: nc.MustGetTableName("identity_configurations"),
		IdentityInfo:           nc.MustGetTableName("identity_information"),
		Signers:                nc.MustGetTableName("identity_signers"),
	}, nil
}
