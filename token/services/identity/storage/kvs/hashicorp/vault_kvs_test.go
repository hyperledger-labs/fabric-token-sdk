/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs_test

import (
	"fmt"
	"sync"
	"testing"

	vault "github.com/hashicorp/vault/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	hashicorp "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs/hashicorp"
	"github.com/stretchr/testify/assert"
)

type stuff struct {
	S string `json:"s"`
	I int    `json:"i"`
}

// creates a new Vault client
func NewVaultClient(address, token string) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = address

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %v", err)
	}

	client.SetToken(token)

	return client, nil
}

func testRound(t *testing.T, client *vault.Client) {
	kvstore, err := hashicorp.NewWithClient(client, "secret/data/token-sdk/")
	assert.NoError(t, err)

	k1, err := kvs.CreateCompositeKey("k", []string{"1"})
	assert.NoError(t, err)
	k2, err := kvs.CreateCompositeKey("k", []string{"2"})
	assert.NoError(t, err)

	err = kvstore.Put(k1, &stuff{"santa", 1})
	assert.NoError(t, err)

	val := &stuff{}
	err = kvstore.Get(k1, val)
	assert.NoError(t, err)
	assert.Equal(t, &stuff{"santa", 1}, val)

	err = kvstore.Put(k2, &stuff{"claws", 2})
	assert.NoError(t, err)

	val = &stuff{}
	err = kvstore.Get(k2, val)
	assert.NoError(t, err)
	assert.Equal(t, &stuff{"claws", 2}, val)

	it, err := kvstore.GetByPartialCompositeID("k", []string{})
	assert.NoError(t, err)
	defer it.Close()

	for ctr := 0; it.HasNext(); ctr++ {
		val = &stuff{}
		key, err := it.Next(val)
		assert.NoError(t, err)
		if ctr == 0 {
			assert.Equal(t, k1, key)
			assert.Equal(t, &stuff{"santa", 1}, val)
		} else if ctr == 1 {
			assert.Equal(t, k2, key)
			assert.Equal(t, &stuff{"claws", 2}, val)
		} else {
			assert.Fail(t, "expected 2 entries in the range, found more")
		}
	}

	//assert.NoError(t, kvstore.Delete(k2)) TBD
	assert.False(t, kvstore.Exists(k2))
	val = &stuff{}
	err = kvstore.Get(k2, val)
	assert.Error(t, err)

	for ctr := 0; it.HasNext(); ctr++ {
		val = &stuff{}
		key, err := it.Next(val)
		assert.NoError(t, err)
		if ctr == 0 {
			assert.Equal(t, k1, key)
			assert.Equal(t, &stuff{"santa", 1}, val)
		} else {
			assert.Fail(t, "expected 2 entries in the range, found more")
		}
	}

	val = &stuff{
		S: "hello",
		I: 100,
	}
	k := hash.Hashable("Hello World").RawString()
	assert.NoError(t, kvstore.Put(k, val))
	assert.True(t, kvstore.Exists(k))
	val2 := &stuff{}
	assert.NoError(t, kvstore.Get(k, val2))
	assert.Equal(t, val, val2)
	//assert.NoError(t, kvstore.Delete(k)) TBD
	assert.False(t, kvstore.Exists(k))
}

func testParallelWrites(t *testing.T, client *vault.Client) {
	kvstore, err := hashicorp.NewWithClient(client, "secret/data/token-sdk/")
	assert.NoError(t, err)
	// defer kvstore.Stop() TBD

	// different keys
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			k1, err := kvs.CreateCompositeKey("parallel_key_1_", []string{fmt.Sprintf("%d", i)})
			assert.NoError(t, err)
			err = kvstore.Put(k1, &stuff{"santa", i})
			assert.NoError(t, err)
			defer wg.Done()
		}(i)
	}
	wg.Wait()

	// same key
	wg = sync.WaitGroup{}
	wg.Add(n)
	k1, err := kvs.CreateCompositeKey("parallel_key_2_", []string{"1"})
	assert.NoError(t, err)
	for i := 0; i < n; i++ {
		go func(i int) {
			err := kvstore.Put(k1, &stuff{"santa", 1})
			assert.NoError(t, err)
			defer wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestVaultKVS(t *testing.T) {
	client, err := NewVaultClient("http://127.0.0.1:8200", "root")
	assert.NoError(t, err)

	testRound(t, client)
	testParallelWrites(t, client)
}
