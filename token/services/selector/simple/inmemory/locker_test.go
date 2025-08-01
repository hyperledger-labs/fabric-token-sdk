/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package inmemory

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestLockEntry(t *testing.T) {
	m := map[token.ID]string{}

	id1 := token.ID{
		TxId:  "a",
		Index: 0,
	}
	id2 := token.ID{
		TxId:  "a",
		Index: 0,
	}

	m[id1] = "a"
	m[id2] = "b"
	assert.Len(t, m, 1)
	assert.Equal(t, "b", m[id1])
	assert.Equal(t, "b", m[id2])
}
