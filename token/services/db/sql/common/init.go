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

var tnc = db.NewTableNameCreatorWithDefaultPrefix("fts")

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
	nc, err := tnc.GetFormatter(prefix)
	if err != nil {
		return tableNames{}, err
	}

	return tableNames{
		Movements:              nc.MustFormat("movements", params...),
		Transactions:           nc.MustFormat("txs", params...),
		TransactionEndorseAck:  nc.MustFormat("tx_ends", params...),
		Requests:               nc.MustFormat("requests", params...),
		Validations:            nc.MustFormat("req_vals", params...),
		Tokens:                 nc.MustFormat("tokens", params...),
		Ownership:              nc.MustFormat("tkn_own", params...),
		Certifications:         nc.MustFormat("tkn_crts", params...),
		PublicParams:           nc.MustFormat("public_params", params...),
		Wallets:                nc.MustFormat("wallets", params...),
		IdentityConfigurations: nc.MustFormat("id_cfgs", params...),
		IdentityInfo:           nc.MustFormat("id_info", params...),
		Signers:                nc.MustFormat("id_signers", params...),
		TokenLocks:             nc.MustFormat("tkn_locks", params...),
	}, nil
}
