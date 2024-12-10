/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sig

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/stretchr/testify/assert"
)

type Signer struct {
	ID string
}

func (s *Signer) Sign(message []byte) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

type Verifier struct {
	ID string
}

func (v *Verifier) Verify(message, sigma []byte) error {

	// TODO implement me
	panic("implement me")
}

func BenchmarkRegisterSigner(b *testing.B) {
	b.StopTimer()
	terminate, pgConnStr, err := postgres.StartPostgres(b, false)
	assert.NoError(b, err)
	defer terminate()
	d := &postgres.TestDriver{
		Name:    "hw",
		ConnStr: pgConnStr,
	}
	registry := registry.New()
	cp := &mock.ConfigProvider{}
	backend, err := kvs.NewWithConfig(d, "", cp)
	assert.NoError(b, err)
	err = registry.RegisterService(backend)
	assert.NoError(b, err)
	sigService := NewService(NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		// Code to be benchmarked
		b.StopTimer()
		nonce, err := htlc.CreateNonce()
		assert.NoError(b, err)
		signer := &Signer{ID: string(nonce)}
		verifier := &Verifier{ID: string(nonce)}
		b.StartTimer()

		assert.NoError(b, sigService.RegisterSigner(nonce, signer, verifier, nil))
	}
}
