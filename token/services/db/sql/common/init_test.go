/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/test-go/testify/assert"
	_ "modernc.org/sqlite"
)

func TestGetTableNames(t *testing.T) {
	names, err := GetTableNames("")
	assert.NoError(t, err)
	assert.Equal(t, tableNames{
		Movements:              "fts_movements",
		Transactions:           "fts_txs",
		TransactionEndorseAck:  "fts_tx_ends",
		Requests:               "fts_requests",
		Validations:            "fts_req_vals",
		Tokens:                 "fts_tokens",
		Ownership:              "fts_tkn_own",
		Certifications:         "fts_tkn_crts",
		PublicParams:           "fts_public_params",
		Wallets:                "fts_wallets",
		IdentityConfigurations: "fts_id_cfgs",
		IdentityInfo:           "fts_id_info",
		Signers:                "fts_id_signers",
		TokenLocks:             "fts_tkn_locks",
	}, names)

	names, err = GetTableNames("valid_prefix")
	assert.NoError(t, err)
	assert.Equal(t, "valid_prefix_txs", names.Transactions)

	names, err = GetTableNames("Valid_Prefix")
	assert.NoError(t, err)
	assert.Equal(t, "valid_prefix_txs", names.Transactions)

	names, err = GetTableNames("valid")
	assert.NoError(t, err)
	assert.Equal(t, "valid_txs", names.Transactions)

	invalid := []string{
		"invalid;",
		"invalid ",
		"in<valid",
		"in\\valid",
		"in\bvalid",
		"invalid\x00",
		"\"invalid\"",
		"in_valid1",
		"Invalid-Prefix",
		"too_long_abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij",
	}

	for _, inv := range invalid {
		t.Run(fmt.Sprintf("Prefix: %s", inv), func(t *testing.T) {
			names, err := GetTableNames(inv)
			assert.Error(t, err)
			assert.Equal(t, tableNames{}, names)
		})
	}
}
