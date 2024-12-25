/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/stretchr/testify/assert"
)

func TestSelect_Compile(t *testing.T) {
	// Simple SELECT
	selectStmt := common.NewSelect("id", "name").From("users").Where("id = 1")
	query, err := selectStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT id,name FROM users WHERE id = 1", query)

	selectStmt = common.NewSelect("id", "name").From("users", "citizen").Where("id = 1")
	query, err = selectStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT id,name FROM users citizen WHERE id = 1", query)

	// SELECT DISTINCT
	selectDistinctStmt := common.NewSelectDistinct("email").From("customers")
	query, err = selectDistinctStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT DISTINCT email FROM customers", query)

	// No columns selected
	selectStmt = common.NewSelect().From("users")
	query, err = selectStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM users", query)
}

func TestInsert_Compile(t *testing.T) {
	// Simple INSERT
	insertStmt := common.NewInsertInto("users").Rows("id, name")
	query, err := insertStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO users (id, name) VALUES ($1, $2)", query)

	// Missing rows
	insertStmt = common.NewInsertInto("users")
	_, err = insertStmt.Compile()
	assert.Error(t, err)
}

func TestUpdate_Compile(t *testing.T) {
	// Simple UPDATE
	updateStmt := common.NewUpdate("users").Set("name, age").Where("id")
	query, err := updateStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "UPDATE users SET name = $1, age = $2 WHERE id = $3", query)

	// Missing table
	updateStmt = common.NewUpdate("").Set("name")
	_, err = updateStmt.Compile()
	assert.Error(t, err)
}

func TestDelete_Compile(t *testing.T) {
	// Simple DELETE
	deleteStmt := common.NewDeleteFrom("users").Where("id = 1")
	query, err := deleteStmt.Compile()
	assert.NoError(t, err)
	assert.Equal(t, "DELETE FROM users WHERE id = 1", query)

	// Missing table
	deleteStmt = common.NewDeleteFrom("").Where("id = 1")
	_, err = deleteStmt.Compile()
	assert.Error(t, err)
}
