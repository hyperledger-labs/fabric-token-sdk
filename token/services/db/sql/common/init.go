/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/db"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

var ncProvider = db.NewTableNameCreator("fsc")

type TableNames struct {
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

type PersistenceConstructor[V common.DBObject] func(*common.RWDB, TableNames) (V, error)

func GetTableNames(prefix string, params ...string) (TableNames, error) {
	nc, err := ncProvider.GetFormatter(prefix)
	if err != nil {
		return TableNames{}, err
	}

	return TableNames{
		Movements:              nc.MustFormat("movements", params...),
		Transactions:           nc.MustFormat("txs", params...),
		TransactionEndorseAck:  nc.MustFormat("tx_ends", params...),
		Requests:               nc.MustFormat("requests", params...),
		Validations:            nc.MustFormat("req_vals", params...),
		Tokens:                 nc.MustFormat("tokens", params...),
		Ownership:              nc.MustFormat("tkn_own", params...),
		Certifications:         nc.MustFormat("tkn_crts", params...),
		TokenLocks:             nc.MustFormat("tkn_locks", params...),
		PublicParams:           nc.MustFormat("public_params", params...),
		Wallets:                nc.MustFormat("wallets", params...),
		IdentityConfigurations: nc.MustFormat("id_cfgs", params...),
		IdentityInfo:           nc.MustFormat("id_info", params...),
		Signers:                nc.MustFormat("id_signers", params...),
	}, nil
}
