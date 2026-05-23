/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package _insert_test

import (
	"testing"

	. "github.com/onsi/gomega"

	q "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
)

func TestInsertSimple(t *testing.T) { //nolint:paralleltest
	RegisterTestingT(t)

	query, params := q.InsertInto("my_table").
		Fields("key", "data").
		Row("val1", "val2").
		OnConflictDoNothing().
		Format()

	Expect(query).To(Equal("INSERT INTO my_table " +
		"(key, data) " +
		"VALUES ($1, $2) " +
		"ON CONFLICT DO NOTHING"))
	Expect(params).To(ConsistOf("val1", "val2"))
}

func TestInsertOnConflict(t *testing.T) { //nolint:paralleltest
	RegisterTestingT(t)

	query, params := q.InsertInto("my_table").
		Fields("key", "data").
		Row("val1", "val2").
		OnConflict([]common.FieldName{"key", "data"}, q.SetValue("data", "val3"), q.OverwriteValue("key")).
		Format()

	Expect(query).To(Equal("INSERT INTO my_table " +
		"(key, data) " +
		"VALUES ($1, $2) " +
		"ON CONFLICT (key, data) DO UPDATE SET " +
		"data=$3, key=excluded.key"))
	Expect(params).To(ConsistOf("val1", "val2", "val3"))
}
