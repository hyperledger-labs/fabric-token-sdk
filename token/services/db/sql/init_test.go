/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	"github.com/test-go/testify/assert"
	_ "modernc.org/sqlite"
)

func TestGetTableNames(t *testing.T) {
	names, err := getTableNames("")
	assert.NoError(t, err)
	assert.Equal(t, tableNames{
		Movements:              "movements",
		Transactions:           "transactions",
		Requests:               "requests",
		Validations:            "validations",
		TransactionEndorseAck:  "tea",
		Certifications:         "certifications",
		Tokens:                 "tokens",
		Ownership:              "ownership",
		PublicParams:           "public_params",
		Wallets:                "wallet",
		IdentityConfigurations: "id_configs",
		IdentityInfo:           "id_info",
		Signers:                "signers",
	}, names)

	names, err = getTableNames("valid_prefix")
	assert.NoError(t, err)
	assert.Equal(t, "valid_prefix_transactions", names.Transactions)

	names, err = getTableNames("Valid_Prefix")
	assert.NoError(t, err)
	assert.Equal(t, "valid_prefix_transactions", names.Transactions)

	names, err = getTableNames("valid")
	assert.NoError(t, err)
	assert.Equal(t, "valid_transactions", names.Transactions)

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
			names, err := getTableNames(inv)
			assert.Error(t, err)
			assert.Equal(t, tableNames{}, names)
		})
	}
}
