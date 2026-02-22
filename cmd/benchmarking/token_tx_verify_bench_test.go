/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/require"
)

// viewPool pre-generates distinct views so that benchmark iterations rotate
// through different ZK proofs instead of repeatedly verifying same data.
type viewPool struct {
	views []view.View
	size  int
	idx   int
}

func (p *viewPool) createViews(b *testing.B, f *TokenTxVerifyViewFactory, input []byte, n int) {
	b.Helper()
	p.views = make([]view.View, n)
	for i := range p.views {
		v, err := f.NewView(input)
		require.NoError(b, err)
		p.views[i] = v
	}
	p.size = n
	p.idx = 0
}

// nextView returns views from the pool in round-robin.
func (p *viewPool) nextView() view.View {
	i := p.idx
	p.idx = (p.idx + 1) % p.size

	return p.views[i%p.size]
}

func BenchmarkTokenTxVerify(b *testing.B) {
	p := &TokenTxVerifyMetadata{}
	p.applyDefaults()

	b.Run(fmt.Sprintf("out-tokens=%d", p.NumOutputTokens), func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}
		input, _ := json.Marshal(p)

		pool := &viewPool{}
		pool.createViews(b, f, input, max(runtime.NumCPU()*4, 16))

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = pool.nextView().Call(nil)
			}
		})
		benchmark.ReportTPS(b)
	})
}

func BenchmarkTokenTxVerify_PreComputeProof(b *testing.B) {
	p := &TokenTxVerifyMetadata{}
	p.applyDefaults()

	b.Run(fmt.Sprintf("out-tokens=%d", p.NumOutputTokens), func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}
		input, _ := json.Marshal(p)

		proofs := make([]*ProofData, b.N)
		for i := range proofs {
			proof, err := GenerateProofData(p)
			require.NoError(b, err)
			proofs[i] = proof
		}
		f.AddProofs(proofs...)

		pool := &viewPool{}
		pool.createViews(b, f, input, max(runtime.NumCPU()*4, 16))

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = pool.nextView().Call(nil)
			}
		})
		benchmark.ReportTPS(b)
	})
}
