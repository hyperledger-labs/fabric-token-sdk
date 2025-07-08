/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashicorp_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"testing"

	vault "github.com/hashicorp/vault/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs/hashicorp"
	"github.com/stretchr/testify/assert"
)

type stuff struct {
	S string `json:"s"`
	I int    `json:"i"`
}

func TestVaultKVS(t *testing.T) {
	terminate, vaultURL, token := hashicorp.StartHashicorpVaultContainer(t, 10200)
	defer terminate()
	client, err := hashicorp.NewVaultClient(vaultURL, token)
	assert.NoError(t, err)

	testRound(t, client)
	testParallelWrites(t, client)
	testParallelWritesReadDelete(t, client)
	testParallelConnections(t, client)

	terminate()

	testWithVaultDown(t, client)
}

func testRound(t *testing.T, client *vault.Client) {
	// Test with slah at the end of the vault path
	kvstore, err := hashicorp.NewWithClient(client, "kv1/data/token-sdk/")
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

	results := kvstore.GetExisting(k1, k2)
	assert.True(t, len(results) == 2)

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

	assert.NoError(t, kvstore.Delete(k2))
	assert.False(t, kvstore.Exists(k2))

	results = kvstore.GetExisting(k1, k2)
	assert.True(t, len(results) == 1)
	assert.True(t, results[0] == k1)

	val = &stuff{}
	err = kvstore.Get(k2, val)
	assert.NoError(t, err)

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

	// Test the iterator calling Next without hasNext first in case the
	// iterator has been exhausted
	_, err = it.Next(val)
	assert.Error(t, err)

	it, err = kvstore.GetByPartialCompositeID("k", []string{})
	assert.NoError(t, err)
	defer it.Close()
	for ctr := 0; it.HasNext(); ctr++ {
		val = &stuff{}
		key, err := it.Next(val)
		assert.NoError(t, err)
		if ctr == 0 {
			assert.Equal(t, k1, key)
			assert.Equal(t, &stuff{"santa", 1}, val)
		} else {
			assert.Fail(t, "expected 1 entries in the range, found more")
		}
	}

	assert.NoError(t, kvstore.Delete(k1))

	val = &stuff{
		S: "hello",
		I: 100,
	}
	data := "Hello World"
	hash := sha256.Sum256([]byte(data)) // Replace with hash.Hashable if applicable
	k := hex.EncodeToString(hash[:])    // Convert to clean hex string

	assert.NoError(t, kvstore.Put(k, val))
	assert.True(t, kvstore.Exists(k))
	val2 := &stuff{}
	assert.NoError(t, kvstore.Get(k, val2))
	assert.Equal(t, val, val2)

	results = kvstore.GetExisting(k)
	assert.True(t, len(results) == 1)

	it, err = kvstore.GetByPartialCompositeID(k, []string{})
	assert.NoError(t, err)
	assert.True(t, it == nil)
	assert.NoError(t, kvstore.Delete(k))
	assert.False(t, kvstore.Exists(k))

	k1, err = kvs.CreateCompositeKey(k, []string{"1"})
	assert.NoError(t, err)
	assert.NoError(t, kvstore.Put(k1, val))
	it, err = kvstore.GetByPartialCompositeID(k, []string{})
	assert.NoError(t, err)
	defer it.Close()
	for ctr := 0; it.HasNext(); ctr++ {
		val = &stuff{}
		key, err := it.Next(val)
		assert.NoError(t, err)
		if ctr == 0 {
			assert.Equal(t, k1, key)
			assert.Equal(t, &stuff{"hello", 100}, val)
		} else {
			assert.Fail(t, "expected 1 entries in the range, found more")
		}
	}
	assert.NoError(t, kvstore.Delete(k1))
	assert.False(t, kvstore.Exists(k1))
	assert.True(t, kvstore.Delete(k1) == nil)

	it, err = kvstore.GetByPartialCompositeID(k, []string{})
	assert.NoError(t, err)
	assert.True(t, it == nil)

	_, err = kvstore.GetByPartialCompositeID("k", []string{})
	assert.NoError(t, err)

	k3, err := kvs.CreateCompositeKey("k", []string{"3"})
	assert.NoError(t, err)

	err = kvstore.Put(k3, nil)
	assert.NoError(t, err)

	err = kvstore.Get(k3, nil)
	assert.Error(t, err)

	assert.NoError(t, kvstore.Delete(k3))
	assert.NoError(t, kvstore.Delete(k3))

	err = kvstore.Get(k3, nil)
	assert.NoError(t, err)
	assert.True(t, it == nil)

	k4, _ := kvs.CreateCompositeKey("k", []string{"4"})
	assert.NoError(t, kvstore.Delete(k4))

	results = kvstore.GetExisting()
	assert.True(t, len(results) == 0)
}

func testParallelWrites(t *testing.T, client *vault.Client) {
	kvstore, err := hashicorp.NewWithClient(client, "kv1/data/token-sdk")
	assert.NoError(t, err)

	// different composite key keys
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

	// different none composite key keys
	wg = sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			data := "Hello World " + strconv.Itoa(i)
			hash := sha256.Sum256([]byte(data)) // Replace with hash.Hashable if applicable
			k2 := hex.EncodeToString(hash[:])   // Convert to clean hex string
			err := kvstore.Put(k2, &stuff{"hello", 1})
			assert.NoError(t, err)
			defer wg.Done()
		}(i)
	}
	wg.Wait()
}

func testParallelWritesReadDelete(t *testing.T, client *vault.Client) {
	kvstore, err := hashicorp.NewWithClient(client, "kv1/data/token-sdk")
	assert.NoError(t, err)

	// different composite key keys
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {

			k, err := kvs.CreateCompositeKey("parallel_key_2_", []string{fmt.Sprintf("%d", i)})
			assert.NoError(t, err)

			err = kvstore.Put(k, &stuff{"santa", i})
			assert.NoError(t, err)

			val := &stuff{}
			err = kvstore.Get(k, val)
			assert.NoError(t, err)
			assert.Equal(t, &stuff{"santa", i}, val)

			assert.NoError(t, kvstore.Delete(k))
			defer wg.Done()
		}(i)
	}
	wg.Wait()
}

func testClient(t *testing.T, wg *sync.WaitGroup, prefix string, num int, client *vault.Client) {
	defer wg.Done()

	// Test without slah at the end of the vault path
	kvstore, err := hashicorp.NewWithClient(client, "kv1/data/token-sdk")
	assert.NoError(t, err)

	for i := 1; i <= num; i++ {
		k, err := kvs.CreateCompositeKey(prefix, []string{fmt.Sprintf("%d", i)})
		assert.NoError(t, err)

		err = kvstore.Put(k, &stuff{"santa", i})
		assert.NoError(t, err)

		val := &stuff{}
		err = kvstore.Get(k, val)
		assert.NoError(t, err)
		assert.Equal(t, &stuff{"santa", i}, val)

		assert.NoError(t, kvstore.Delete(k))
	}
}

func testParallelConnections(t *testing.T, client *vault.Client) {
	var wg sync.WaitGroup
	// test 20 clients that issues 50 put, get and delete to vault
	n := 20
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go testClient(t, &wg, "parallel_client_"+strconv.Itoa(i), 50, client)
	}
	wg.Wait()
}

func testWithVaultDown(t *testing.T, client *vault.Client) {
	// Test with slah at the end of the vault path
	kvstore, err := hashicorp.NewWithClient(client, "kv1/data/token-sdk/")
	assert.NoError(t, err)

	k1, err := kvs.CreateCompositeKey("k", []string{"1"})
	assert.NoError(t, err)
	k2, err := kvs.CreateCompositeKey("k", []string{"2"})
	assert.NoError(t, err)

	err = kvstore.Put(k1, &stuff{"santa", 1})
	assert.Error(t, err)

	val := &stuff{}
	err = kvstore.Get(k1, val)
	assert.Error(t, err)

	assert.False(t, kvstore.Exists(k2))

	results := kvstore.GetExisting(k1, k2)
	assert.True(t, len(results) == 0)

	assert.Error(t, kvstore.Delete(k1))

	it, err := kvstore.GetByPartialCompositeID("k", []string{})
	assert.Error(t, err)
	assert.True(t, it == nil)
}
