/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"testing"

	math "github.com/IBM/mathlib"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestSerialization(t *testing.T) {
	action := randomAction(math.Curves[math.BN254], rand.Reader, t)
	raw, err := action.Serialize()
	assert.NoError(t, err, "failed to serialize a new transfer action")

	action2 := &Action{}
	err = action2.Deserialize(raw)
	assert.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action, action2, "deserialized action is not equal to the original one")

	raw2, err := action2.Serialize()
	assert.NoError(t, err, "failed to serialize a new transfer action")

	action3 := &Action{}
	err = action3.Deserialize(raw2)
	assert.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action2, action3, "deserialized action is not equal to the original one")
}

func BenchmarkActionMarshalling(b *testing.B) {
	curve := math.Curves[math.BN254]

	b.Run("With Protos", func(b *testing.B) {
		rand, err := curve.Rand()
		assert.NoError(b, err, "failed to get random number")
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			action := randomAction(curve, rand, b)
			b.StartTimer()
			_, err = action.Serialize()
			assert.NoError(b, err, "failed to serialize a new transfer action")
		}
	})

	b.Run("With json", func(b *testing.B) {
		rand, err := curve.Rand()
		assert.NoError(b, err, "failed to get random number")
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			action := randomAction(curve, rand, b)
			b.StartTimer()
			_, err = json.Marshal(action)
			assert.NoError(b, err, "failed to serialize a new transfer action")
		}
	})
}

func getRandomBytes(b assert.TestingT, len int) []byte {
	key := make([]byte, len)
	_, err := rand.Read(key)
	assert.NoError(b, err, "error getting random bytes")
	return key
}

func randomAction(curve *math.Curve, rand io.Reader, b assert.TestingT) *Action {
	// generate an action at random
	issuer := getRandomBytes(b, 32)
	commitments := []*math.G1{
		curve.GenG1.Mul(curve.NewRandomZr(rand)),
		curve.GenG1.Mul(curve.NewRandomZr(rand)),
	}
	owners := [][]byte{
		getRandomBytes(b, 32),
		getRandomBytes(b, 32),
	}
	proof := getRandomBytes(b, 32)
	action, err := NewAction(issuer, commitments, owners, proof)
	assert.NoError(b, err, "failed to create a new transfer action")
	action.Inputs = []*ActionInput{
		{ID: token2.ID{
			TxId:  "txid1",
			Index: 0,
		}, Token: curve.GenG1.Mul(curve.NewRandomZr(rand)).Bytes()},
		{ID: token2.ID{
			TxId:  "txid2",
			Index: 1,
		}, Token: curve.GenG1.Mul(curve.NewRandomZr(rand)).Bytes()},
	}
	action.Metadata = map[string][]byte{
		"key1": getRandomBytes(b, 32),
		"key2": getRandomBytes(b, 32),
	}
	return action
}
