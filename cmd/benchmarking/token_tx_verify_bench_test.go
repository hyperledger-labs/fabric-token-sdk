/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/require"
)

// viewPool pre-generates distinct views so that benchmark iterations rotate
// through different ZK proofs instead of repeatedly verifying same data.

const (
	defaultNumViews = 16
)

type viewPool struct {
	views []view.View
	idx   atomic.Int64
}

func CreateViewsWithProofs(b *testing.B, p *TokenTxVerifyParams, f *TokenTxVerifyViewFactory, n int) *viewPool {
	b.Helper()

	vp := &viewPool{}

	// Create n views
	vp.views = make([]view.View, n)
	for i := range vp.views {
		// create proof
		proof, err := GenerateProofData(p)
		require.NoError(b, err)

		// create view
		p.Proof, _ = proof.ToWire()
		input, _ := json.Marshal(p)
		v, err := f.NewView(input)
		require.NoError(b, err)
		vp.views[i] = v
	}

	vp.idx.Store(0)

	return vp
}

// nextView returns views from the pool in round-robin.
func (vp *viewPool) nextView() view.View {
	i := vp.idx.Add(1) - 1
	l := len(vp.views)

	return vp.views[i%int64(l)]
}

func BenchmarkTokenTxVerify(b *testing.B) {
	p := &TokenTxVerifyParams{}
	p.applyDefaults()

	b.Run(fmt.Sprintf("out-tokens=%d", p.NumOutputTokens), func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}

		// We pre instantite a bunch of views, each with a different proof and public params.
		pool := CreateViewsWithProofs(b, p, f, defaultNumViews)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = pool.nextView().Call(nil)
			}
		})
		benchmark.ReportTPS(b)
	})
}
