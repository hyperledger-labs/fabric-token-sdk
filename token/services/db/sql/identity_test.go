/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"path"
	"reflect"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/stretchr/testify/assert"
)

func initIdentityDB(driverName, dataSourceName, tablePrefix string, maxOpenConns int) (*IdentityDB, error) {
	d := NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	return NewIdentityDB(sqlDB, tablePrefix, true, secondcache.New(1000))
}

func TestIdentitySqlite(t *testing.T) {
	tempDir := t.TempDir()

	for _, c := range IdentityCases {
		db, err := initIdentityDB("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestIdentitySqliteMemory(t *testing.T) {
	for _, c := range IdentityCases {
		db, err := initIdentityDB("sqlite", "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared", c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestIdentityPostgres(t *testing.T) {
	terminate, pgConnStr := StartPostgresContainer(t)
	defer terminate()

	for _, c := range IdentityCases {
		db, err := initIdentityDB("pgx", pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

var IdentityCases = []struct {
	Name string
	Fn   func(*testing.T, *IdentityDB)
}{
	{"IdentityInfo", TIdentityInfo},
	{"SignerInfo", TSignerInfo},
	{"Configurations", TConfigurations},
	{"SignerInfoConcurrent", TSignerInfoConcurrent},
}

func TConfigurations(t *testing.T, db *IdentityDB) {
	expected := driver.IdentityConfiguration{
		ID:     "pineapple",
		Type:   "core",
		URL:    "look here",
		Config: []byte("config"),
		Raw:    []byte("raw"),
	}
	assert.NoError(t, db.AddConfiguration(expected))

	it, err := db.IteratorConfigurations("core")
	assert.NoError(t, err)
	assert.True(t, it.HasNext())
	c, err := it.Next()
	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(expected, c))
	assert.NoError(t, it.Close())

	exists, err := db.ConfigurationExists("pineapple", "core")
	assert.NoError(t, err)
	assert.True(t, exists)

	_, err = db.IteratorConfigurations("no core")
	assert.NoError(t, err)
	assert.False(t, it.HasNext())

	exists, err = db.ConfigurationExists("pineapple", "no core")
	assert.NoError(t, err)
	assert.False(t, exists)

	expected = driver.IdentityConfiguration{
		ID:     "pineapple",
		Type:   "no core",
		URL:    "look here",
		Config: []byte("config"),
		Raw:    []byte("raw"),
	}
	assert.NoError(t, db.AddConfiguration(expected))
}

func TIdentityInfo(t *testing.T, db *IdentityDB) {
	id := []byte("alice")
	auditInfo := []byte("alice_audit_info")
	tokMeta := []byte("tok_meta")
	tokMetaAudit := []byte("tok_meta_audit")
	assert.NoError(t, db.StoreIdentityData(id, auditInfo, tokMeta, tokMetaAudit))

	auditInfo2, err := db.GetAuditInfo(id)
	assert.NoError(t, err, "failed to retrieve audit info for [%s]", id)
	assert.Equal(t, auditInfo, auditInfo2)

	tokMeta2, tokMetaAudit2, err := db.GetTokenInfo(id)
	assert.NoError(t, err, "failed to retrieve token info for [%s]", id)
	assert.Equal(t, tokMeta, tokMeta2)
	assert.Equal(t, tokMetaAudit, tokMetaAudit2)
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

func TSignerInfoConcurrent(t *testing.T, db *IdentityDB) {
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			alice := []byte(fmt.Sprintf("alice_%d", i))
			bob := []byte(fmt.Sprintf("bob_%d", i))
			assert.NoError(t, db.StoreSignerInfo(alice, nil))
			exists, err := db.SignerInfoExists(alice)
			assert.NoError(t, err, "failed to check signer info existence for [%s]", alice)
			assert.True(t, exists)

			t.Log(i)
			exists, err = db.SignerInfoExists(bob)
			assert.NoError(t, err, "failed to check signer info existence for [%s]", bob)
			assert.False(t, exists)
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		alice := []byte(fmt.Sprintf("alice_%d", i))
		exists, err := db.SignerInfoExists(alice)
		assert.NoError(t, err, "failed to check signer info existence for [%s]", alice)
		assert.True(t, exists)
	}
}
