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
	assert.Equal(t, TableNames{
		Movements:              "fsc_movements",
		Transactions:           "fsc_txs",
		Requests:               "fsc_requests",
		Validations:            "fsc_req_vals",
		TransactionEndorseAck:  "fsc_tx_ends",
		Certifications:         "fsc_tkn_crts",
		Tokens:                 "fsc_tokens",
		Ownership:              "fsc_tkn_own",
		PublicParams:           "fsc_public_params",
		Wallets:                "fsc_wallets",
		IdentityConfigurations: "fsc_id_cfgs",
		IdentityInfo:           "fsc_id_info",
		Signers:                "fsc_id_signers",
		TokenLocks:             "fsc_tkn_locks",
		KeyStore:               "fsc_keystore",
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
			assert.Equal(t, TableNames{}, names)
		})
	}
}
