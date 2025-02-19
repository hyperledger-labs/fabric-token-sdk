/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"

	math "github.com/IBM/mathlib"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

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

func getRandomBytes(b *testing.B, len int) []byte {
	key := make([]byte, len)
	_, err := rand.Read(key)
	assert.NoError(b, err, "error getting random bytes")
	return key
}

func randomAction(curve *math.Curve, rand io.Reader, b *testing.B) *Action {
	// generate an action at random
	tokenIDs := []*token.ID{
		{
			TxId:  base64.StdEncoding.EncodeToString(getRandomBytes(b, 32)),
			Index: 0,
		},
		{
			TxId:  base64.StdEncoding.EncodeToString(getRandomBytes(b, 32)),
			Index: 0,
		},
	}
	inputToken := []*token2.Token{
		{
			Owner: getRandomBytes(b, 32),
			Data:  curve.GenG1.Mul(curve.NewRandomZr(rand)),
		},
		{
			Owner: getRandomBytes(b, 32),
			Data:  curve.GenG1.Mul(curve.NewRandomZr(rand)),
		},
	}
	commitments := []*math.G1{
		curve.GenG1.Mul(curve.NewRandomZr(rand)),
		curve.GenG1.Mul(curve.NewRandomZr(rand)),
	}
	owners := [][]byte{
		getRandomBytes(b, 32),
		getRandomBytes(b, 32),
	}
	proof := getRandomBytes(b, 32)
	action, err := NewTransfer(tokenIDs, inputToken, commitments, owners, proof)
	assert.NoError(b, err, "failed to create a new transfer action")
	return action
}
