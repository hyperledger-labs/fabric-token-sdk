/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	transfer3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SenderEnv struct {
	sender              *transfer3.Sender
	outvalues           []uint64
	owners              [][]byte
	fakeSigningIdentity *mock.SigningIdentity
}

type testingEnv interface {
	Helper()
	Errorf(format string, args ...interface{})
	FailNow()
}

func NewSenderEnv(t testingEnv, pp *v1.PublicParams) *SenderEnv {
	t.Helper()
	var (
		fakeSigningIdentity *mock.SigningIdentity
		signers             []driver.Signer

		sender *transfer3.Sender

		invalues  []*math.Zr
		outvalues []uint64
		inBF      []*math.Zr
		tokens    []*token.Token

		owners [][]byte
		ids    []*token2.ID
	)
	var err error
	if pp == nil {
		pp, err = v1.Setup(32, nil, TestCurve)
		require.NoError(t, err)
	}
	owners = make([][]byte, 2)
	owners[0] = []byte("bob")
	owners[1] = []byte("charlie")
	signers = make([]driver.Signer, 3)
	fakeSigningIdentity = &mock.SigningIdentity{}
	signers[0] = fakeSigningIdentity
	signers[1] = fakeSigningIdentity
	signers[2] = fakeSigningIdentity

	fakeSigningIdentity.SignReturnsOnCall(0, []byte("signer[0]"), nil)
	fakeSigningIdentity.SignReturnsOnCall(1, []byte("signer[1]"), nil)
	fakeSigningIdentity.SignReturnsOnCall(2, []byte("signer[2]"), nil)

	c := math.Curves[pp.Curve]
	invalues = make([]*math.Zr, 3)
	invalues[0] = c.NewZrFromInt(50)
	invalues[1] = c.NewZrFromInt(20)
	invalues[2] = c.NewZrFromInt(30)

	inBF = make([]*math.Zr, 3)
	rand, err := c.Rand()
	require.NoError(t, err)
	for i := range 3 {
		inBF[i] = c.NewRandomZr(rand)
	}
	outvalues = make([]uint64, 2)
	outvalues[0] = 65
	outvalues[1] = 35

	ids = make([]*token2.ID, 3)
	ids[0] = &token2.ID{TxId: "0"}
	ids[1] = &token2.ID{TxId: "1"}
	ids[2] = &token2.ID{TxId: "3"}

	inputs := PrepareTokens(invalues, inBF, "ABC", pp.PedersenGenerators, c)
	tokens = make([]*token.Token, 3)

	tokens[0] = &token.Token{Data: inputs[0], Owner: []byte("alice-1")}
	tokens[1] = &token.Token{Data: inputs[1], Owner: []byte("alice-2")}
	tokens[2] = &token.Token{Data: inputs[2], Owner: []byte("alice-3")}

	inputInf := make([]*token.Metadata, 3)
	inputInf[0] = &token.Metadata{Type: "ABC", Value: invalues[0], BlindingFactor: inBF[0]}
	inputInf[1] = &token.Metadata{Type: "ABC", Value: invalues[1], BlindingFactor: inBF[1]}
	inputInf[2] = &token.Metadata{Type: "ABC", Value: invalues[2], BlindingFactor: inBF[2]}

	sender, err = transfer3.NewSender(signers, tokens, ids, inputInf, pp)
	require.NoError(t, err)

	return &SenderEnv{
		sender:              sender,
		outvalues:           outvalues,
		owners:              owners,
		fakeSigningIdentity: fakeSigningIdentity,
	}
}

func TestSender(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := NewSenderEnv(t, nil)

		transfer, _, err := env.sender.GenerateZKTransfer(t.Context(), env.outvalues, env.owners)
		require.NoError(t, err)
		assert.NotNil(t, transfer)
		raw, err := transfer.Serialize()
		require.NoError(t, err)

		sig, err := env.sender.SignTokenActions(raw)
		require.NoError(t, err)
		assert.Equal(t, 3, env.fakeSigningIdentity.SignCallCount())
		assert.Len(t, sig, 3)
	})

	t.Run("when signature fails", func(t *testing.T) {
		env := NewSenderEnv(t, nil)
		env.fakeSigningIdentity.SignReturnsOnCall(2, nil, errors.New("banana republic"))
		transfer, _, err := env.sender.GenerateZKTransfer(t.Context(), env.outvalues, env.owners)
		require.NoError(t, err)
		assert.NotNil(t, transfer)
		raw, err := transfer.Serialize()
		require.NoError(t, err)

		sig, err := env.sender.SignTokenActions(raw)
		require.Error(t, err)
		assert.Nil(t, sig)
		assert.Equal(t, 3, env.fakeSigningIdentity.SignCallCount())
		assert.Contains(t, err.Error(), "banana republic")
	})
}

func PrepareTokens(values, bf []*math.Zr, ttype string, pp []*math.G1, curve *math.Curve) []*math.G1 {
	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = prepareToken(values[i], bf[i], ttype, pp, curve)
	}
	return tokens
}

type BenchmarkSenderEnv struct {
	SenderEnvs []*SenderEnv
}

func NewBenchmarkSenderEnv(b *testing.B, n int) *BenchmarkSenderEnv {
	b.Helper()
	envs := make([]*SenderEnv, n)
	for i := range envs {
		envs[i] = NewSenderEnv(b, nil)
	}
	return &BenchmarkSenderEnv{SenderEnvs: envs}
}

func BenchmarkSender(b *testing.B) {
	b.ReportAllocs()

	// prepare env
	env := NewBenchmarkSenderEnv(b, b.N)

	// Optional: Reset timer if you had expensive setup code above
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		transfer, _, err := env.SenderEnvs[i].sender.GenerateZKTransfer(
			b.Context(),
			env.SenderEnvs[i].outvalues,
			env.SenderEnvs[i].owners,
		)
		require.NoError(b, err)
		assert.NotNil(b, transfer)
		_, err = transfer.Serialize()
		require.NoError(b, err)
	}
}
