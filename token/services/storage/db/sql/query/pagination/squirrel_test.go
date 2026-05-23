/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pagination_test

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/pagination"
)

func baseQuery() sq.SelectBuilder {
	return sq.Select("tx_id", "code").From("status")
}

func TestApplyToSquirrel_Nil(t *testing.T) {
	t.Parallel()
	q, args, err := pagination.ApplyToSquirrel(nil, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_None(t *testing.T) {
	t.Parallel()
	q, args, err := pagination.ApplyToSquirrel(pagination.None(), baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_Empty(t *testing.T) {
	t.Parallel()
	q, args, err := pagination.ApplyToSquirrel(pagination.Empty(), baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status LIMIT 0", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_OffsetPageSizeOnly(t *testing.T) {
	t.Parallel()
	p, err := pagination.Offset(0, 10)
	require.NoError(t, err)

	q, args, err := pagination.ApplyToSquirrel(p, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status LIMIT 10", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_OffsetAndPageSize(t *testing.T) {
	t.Parallel()
	p, err := pagination.Offset(5, 10)
	require.NoError(t, err)

	q, args, err := pagination.ApplyToSquirrel(p, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status LIMIT 10 OFFSET 5", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_KeysetString_NoFirstID(t *testing.T) {
	t.Parallel()
	p, err := pagination.Keyset[string, any](0, 10, "tx_id", nil)
	require.NoError(t, err)

	q, args, err := pagination.ApplyToSquirrel(p, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status ORDER BY tx_id ASC LIMIT 10", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_KeysetString_WithFirstID(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"offset":0,"page_size":10,"sqlid_name":"tx_id","first_id":"abc","last_id":""}`)
	p, err := pagination.KeysetFromRaw[string](raw, "TxID")
	require.NoError(t, err)

	q, args, err := pagination.ApplyToSquirrel(p, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status WHERE tx_id > ? ORDER BY tx_id ASC LIMIT 10", q)
	require.Equal(t, []any{"abc"}, args)
}

func TestApplyToSquirrel_KeysetString_WithOffset(t *testing.T) {
	t.Parallel()
	p, err := pagination.Keyset[string, any](5, 10, "tx_id", nil)
	require.NoError(t, err)

	q, args, err := pagination.ApplyToSquirrel(p, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status ORDER BY tx_id ASC LIMIT 10 OFFSET 5", q)
	require.Empty(t, args)
}

func TestApplyToSquirrel_KeysetInt_NoFirstID(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"offset":0,"page_size":10,"sqlid_name":"pos","first_id":-1,"last_id":-1}`)
	p, err := pagination.KeysetFromRaw[int](raw, "Pos")
	require.NoError(t, err)

	q, args, err := pagination.ApplyToSquirrel(p, baseQuery()).ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT tx_id, code FROM status ORDER BY pos ASC LIMIT 10", q)
	require.Empty(t, args)
}
