/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"testing"

	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	"github.com/stretchr/testify/assert"
)

func TestSelect_Compile(t *testing.T) {
	// Simple SELECT
	query, args := q.Select().
		FieldsByName("id", "name").
		From(q.Table("users")).
		Where(cond.Eq("id", 1)).
		Format(sqlite.NewConditionInterpreter())
	assert.Equal(t, "SELECT id, name FROM users WHERE id = $1", query)
	assert.Equal(t, 1, args[0])

	// SELECT DISTINCT
	query, args = q.SelectDistinct().
		FieldsByName("email_id").
		From(q.Table("customers")).
		Format(sqlite.NewConditionInterpreter())
	assert.Equal(t, "SELECT DISTINCT email_id FROM customers", query)
	assert.Empty(t, args)

	// No columns selected
	query, args = q.Select().
		AllFields().
		From(q.Table("users")).
		Format(sqlite.NewConditionInterpreter())
	assert.Equal(t, "SELECT * FROM users", query)
	assert.Empty(t, args)
}

func TestInsert_Compile(t *testing.T) {
	// Simple INSERT
	query, args := q.InsertInto("users").
		Fields("id", "name").
		Row(1, "nnn").
		Format()
	assert.Equal(t, "INSERT INTO users (id, name) VALUES ($1, $2)", query)
	assert.Equal(t, 1, args[0])
	assert.Equal(t, "nnn", args[1])
}

func TestUpdate_Compile(t *testing.T) {
	// Simple UPDATE
	query, args := q.Update("users").
		Set("name", "TheName").
		Set("age", 16).
		Where(cond.Eq("id", 1)).
		Format(sqlite.NewConditionInterpreter())
	assert.Equal(t, "UPDATE users SET name = $1, age = $2 WHERE id = $3", query)
	assert.Equal(t, "TheName", args[0])
	assert.Equal(t, 16, args[1])
	assert.Equal(t, 1, args[2])
}

func TestDelete_Compile(t *testing.T) {
	// Simple DELETE
	query, args := q.DeleteFrom("users").
		Where(cond.Eq("id", 1)).
		Format(sqlite.NewConditionInterpreter())
	assert.Equal(t, "DELETE FROM users WHERE id = $1", query)
	assert.Equal(t, 1, args[0])
}
