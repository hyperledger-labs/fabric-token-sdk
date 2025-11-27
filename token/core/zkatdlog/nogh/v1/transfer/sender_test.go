/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"fmt"
	"strconv"
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

func NewSenderEnv(tb testing.TB, pp *v1.PublicParams, numInputs int, numOutputs int) *SenderEnv {
	tb.Helper()
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
		pp = setup(tb, TestBits, TestCurve)
	}
	signers = make([]driver.Signer, numInputs)
	fakeSigningIdentity = &mock.SigningIdentity{}
	invalues = make([]*math.Zr, numInputs)
	c := math.Curves[pp.Curve]
	inBF = make([]*math.Zr, numInputs)
	ids = make([]*token2.ID, numInputs)
	rand, err := c.Rand()
	require.NoError(tb, err)
	tokens = make([]*token.Token, numInputs)
	inputInf := make([]*token.Metadata, numInputs)

	owners = make([][]byte, numOutputs)
	outvalues = make([]uint64, numOutputs)

	// prepare inputs
	sum := int64(0)
	for i := range numInputs {
		signers[i] = fakeSigningIdentity
		fakeSigningIdentity.SignReturnsOnCall(i, []byte(fmt.Sprintf("signer[%d]", i)), nil)
		v := int64(i*10 + 10)
		sum += v
		invalues[i] = c.NewZrFromInt(v)
		inBF[i] = c.NewRandomZr(rand)
		ids[i] = &token2.ID{TxId: strconv.Itoa(i)}
	}
	inputs := PrepareTokens(invalues, inBF, "ABC", pp.PedersenGenerators, c)

	for i := range numInputs {
		tokens[i] = &token.Token{Data: inputs[i], Owner: []byte(fmt.Sprintf("alice-%d", i))}
		inputInf[i] = &token.Metadata{Type: "ABC", Value: invalues[i], BlindingFactor: inBF[i]}
	}

	outputValue := uint64(sum / int64(numInputs))
	for i := range numOutputs {
		owners[i] = []byte("bob")
		outvalues[i] = outputValue
	}
	// add any adjustment to the last output

	sender, err = transfer3.NewSender(signers, tokens, ids, inputInf, pp)
	require.NoError(tb, err)

	return &SenderEnv{
		sender:              sender,
		outvalues:           outvalues,
		owners:              owners,
		fakeSigningIdentity: fakeSigningIdentity,
	}
}

func TestSender(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := NewSenderEnv(t, nil, 3, 2)

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
		env := NewSenderEnv(t, nil, 3, 2)
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

type BenchmarkSenderCase struct {
	Bits       uint64
	CurveID    math.CurveID
	NumInputs  int
	NumOutputs int
}

// generateBenchmarkCases returns all combinations of BenchmarkSenderCase created
// from the provided slices of bits, curve IDs, number of inputs and outputs.
func generateBenchmarkCases(bits []uint64, curves []math.CurveID, inputs []int, outputs []int) []struct {
	name          string
	benchmarkCase *BenchmarkSenderCase
} {
	var cases []struct {
		name          string
		benchmarkCase *BenchmarkSenderCase
	}
	for _, b := range bits {
		for _, c := range curves {
			for _, ni := range inputs {
				for _, no := range outputs {
					name := fmt.Sprintf("Setup(bits %d, curve %s, #i %d, #o %d)", b, math.CurveIDToString(c), ni, no)
					cases = append(cases, struct {
						name          string
						benchmarkCase *BenchmarkSenderCase
					}{
						name: name,
						benchmarkCase: &BenchmarkSenderCase{
							Bits:       b,
							CurveID:    c,
							NumInputs:  ni,
							NumOutputs: no,
						},
					})
				}
			}
		}
	}
	return cases
}

type BenchmarkSenderEnv struct {
	SenderEnvs []*SenderEnv
}

func NewBenchmarkSenderEnv(b *testing.B, n int, benchmarkCase *BenchmarkSenderCase) *BenchmarkSenderEnv {
	b.Helper()
	envs := make([]*SenderEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = NewSenderEnv(b, pp, benchmarkCase.NumInputs, benchmarkCase.NumOutputs)
	}
	return &BenchmarkSenderEnv{SenderEnvs: envs}
}

func BenchmarkSender(b *testing.B) {
	b.ReportAllocs()

	// Generate test cases programmatically instead of a static literal.
	bits := []uint64{32, 64}
	curves := []math.CurveID{math.BN254, math.BLS12_381_BBS_GURVY}
	inputs := []int{1, 2, 3}
	outputs := []int{1, 2, 3}

	testCases := generateBenchmarkCases(bits, curves, inputs, outputs)

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			env := NewBenchmarkSenderEnv(b, b.N, tc.benchmarkCase)

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
		})
	}
}
