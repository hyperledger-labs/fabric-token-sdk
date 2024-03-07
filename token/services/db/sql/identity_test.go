/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"reflect"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"

	"github.com/stretchr/testify/assert"
)

func TestIdentitySqlite(t *testing.T) {
	tempDir := t.TempDir()

	for _, c := range IdentityCases {
		initSqlite(t, tempDir, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close() // TODO
			c.Fn(xt, Identity)
		})
	}
}

func TestIdentitySqliteMemory(t *testing.T) {
	for _, c := range IdentityCases {
		initSqliteMemory(t, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Identity)
		})
	}
}

func TestIdentityPostgres(t *testing.T) {
	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	for _, c := range IdentityCases {
		initPostgres(t, pgConnStr, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Identity)
		})
	}
}

var IdentityCases = []struct {
	Name string
	Fn   func(*testing.T, *IdentityDB)
}{
	{"TAuditInfo", TAuditInfo},
	{"TSignerInfo", TSignerInfo},
	{"TConfigurations", TConfigurations},
}

func TConfigurations(t *testing.T, db *IdentityDB) {
	expected := driver.IdentityConfiguration{
		ID:   "pineapple",
		Type: "core",
		URL:  "look here",
	}
	assert.NoError(t, db.AddConfiguration(expected))

	it, err := db.IteratorConfigurations("core")
	assert.NoError(t, err)
	assert.True(t, it.HasNext())
	c, err := it.Next()
	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(expected, c))
	assert.NoError(t, it.Close())

	_, err = db.IteratorConfigurations("no core")
	assert.NoError(t, err)
	assert.False(t, it.HasNext())
}

func TAuditInfo(t *testing.T, db *IdentityDB) {
	id := []byte("alice")
	auditInfo := []byte("alice_audit_info")
	assert.NoError(t, db.StoreAuditInfo(id, auditInfo))

	auditInfo2, err := db.GetAuditInfo(id)
	assert.NoError(t, err, "failed to retrieve audit info for [%s]", id)
	assert.Equal(t, auditInfo, auditInfo2)
}

func TSignerInfo(t *testing.T, db *IdentityDB) {
	alice := []byte("alice")
	bob := []byte("bob")
	assert.NoError(t, db.StoreSignerInfo(alice, nil))
	exists, err := db.SignerInfoExists(alice)
	assert.NoError(t, err, "failed to check signer info existence for [%s]", alice)
	assert.True(t, exists)

	exists, err = db.SignerInfoExists(bob)
	assert.NoError(t, err, "failed to check signer info existence for [%s]", bob)
	assert.False(t, exists)
}
