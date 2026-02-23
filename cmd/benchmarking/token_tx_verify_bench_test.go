/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/require"
)

// viewPool pre-generates distinct views so that benchmark iterations rotate
// through different ZK proofs instead of repeatedly verifying same data.
type viewPool struct {
	views []view.View
	size  uint64
	idx   atomic.Uint64
}

func (vp *viewPool) CreateViewsWithProofs(b *testing.B, p *TokenTxVerifyParams, f *TokenTxVerifyViewFactory) {
	b.Helper()
	proofs := make([]*ProofData, b.N)
	for i := range proofs {
		proof, err := GenerateProofData(p)
		require.NoError(b, err)
		proofs[i] = proof
	}

	vp.views = make([]view.View, b.N)
	for i := range vp.views {
		p.Proof, _ = proofs[i].Marshal()
		input, _ := json.Marshal(p)
		v, err := f.NewView(input)
		require.NoError(b, err)
		vp.views[i] = v
	}
	vp.size = uint64(b.N)
	vp.idx.Store(0)
}

func (vp *viewPool) createViewsWithoutProof(b *testing.B, f *TokenTxVerifyViewFactory, input []byte, n uint64) {
	b.Helper()
	vp.views = make([]view.View, n)
	for i := range vp.views {
		v, err := f.NewView(input)
		require.NoError(b, err)
		vp.views[i] = v
	}
	vp.size = n
	vp.idx.Store(0)
}

// nextView returns views from the pool in round-robin.
func (p *viewPool) nextView() view.View {
	i := p.idx.Add(1) - 1
	return p.views[i%p.size]
}

func BenchmarkTokenTxVerify(b *testing.B) {
	p := &TokenTxVerifyParams{}
	p.applyDefaults()

	b.Run(fmt.Sprintf("out-tokens=%d", p.NumOutputTokens), func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}
		input, _ := json.Marshal(p)

		pool := &viewPool{}
		pool.createViewsWithoutProof(b, f, input, uint64(max(runtime.NumCPU()*4, 16)))

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
	p := &TokenTxVerifyParams{}
	p.applyDefaults()

	b.Run(fmt.Sprintf("out-tokens=%d", p.NumOutputTokens), func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}
		pool := &viewPool{}
		pool.CreateViewsWithProofs(b, p, f)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = pool.nextView().Call(nil)
			}
		})
		benchmark.ReportTPS(b)
	})
}
