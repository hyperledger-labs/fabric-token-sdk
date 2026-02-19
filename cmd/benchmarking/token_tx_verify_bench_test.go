/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark"
)

func BenchmarkTokenTxVerify(b *testing.B) {
	p := &TokenTxVerifyParams{}
	name := fmt.Sprintf("out-tokens=%d", p.NumOutputTokens)
	b.Run(name, func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}
		input, _ := json.Marshal(p)

		b.RunParallel(func(pb *testing.PB) {
			v, _ := f.NewView(input)
			for pb.Next() {
				_, _ = v.Call(nil)
			}
		})
		benchmark.ReportTPS(b)
	})
}

func BenchmarkTokenTxVerify_wFactory(b *testing.B) {
	p := &TokenTxVerifyParams{}
	name := fmt.Sprintf("out-tokens=%d", p.NumOutputTokens)
	b.Run(name, func(b *testing.B) {
		f := &TokenTxVerifyViewFactory{}
		input, _ := json.Marshal(p)

		b.RunParallel(func(pb *testing.PB) {
			v, _ := f.NewView(input)
			for pb.Next() {
				_, _ = v.Call(nil)
			}
		})
		benchmark.ReportTPS(b)
	})
}
