/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package boolpolicy

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyInputIDsMatchExpected(t *testing.T) {
	expected := token.ID{TxId: "tx-1", Index: 0}
	other := token.ID{TxId: "tx-2", Index: 0}

	t.Run("single matching input passes", func(t *testing.T) {
		err := verifyInputIDsMatchExpected([]*token.ID{{TxId: "tx-1", Index: 0}}, expected)
		require.NoError(t, err)
	})

	t.Run("single mismatched input is rejected", func(t *testing.T) {
		err := verifyInputIDsMatchExpected([]*token.ID{&other}, expected)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match approved spend request")
	})

	t.Run("any mismatch in a set is rejected", func(t *testing.T) {
		err := verifyInputIDsMatchExpected([]*token.ID{
			{TxId: "tx-1", Index: 0},
			&other,
		}, expected)
		require.Error(t, err)
	})

	t.Run("nil input id is rejected", func(t *testing.T) {
		err := verifyInputIDsMatchExpected([]*token.ID{nil}, expected)
		require.Error(t, err)
	})

	t.Run("empty input list is rejected", func(t *testing.T) {
		err := verifyInputIDsMatchExpected(nil, expected)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no inputs")
	})

	t.Run("differing index alone is rejected", func(t *testing.T) {
		err := verifyInputIDsMatchExpected(
			[]*token.ID{{TxId: "tx-1", Index: 1}},
			token.ID{TxId: "tx-1", Index: 0},
		)
		require.Error(t, err)
	})
}
