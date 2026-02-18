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

var tokenTxCases = []struct {
	numInputs  int
	numOutputs int
}{
	// {1, 1},
	// {1, 2},
	{2, 2},
	// {5, 5},
	// {10, 10},
}

func BenchmarkTokenTxValidate(b *testing.B) {
	for _, tc := range tokenTxCases {
		name := fmt.Sprintf("in=%d/out=%d", tc.numInputs, tc.numOutputs)
		b.Run(name, func(b *testing.B) {
			f := &TokenTxValidateViewFactory{}
			p := &TokenTxValidateParams{NumInputs: tc.numInputs, NumOutputs: tc.numOutputs}
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
}

func BenchmarkTokenTxValidate_wFactory(b *testing.B) {
	for _, tc := range tokenTxCases {
		name := fmt.Sprintf("in=%d/out=%d", tc.numInputs, tc.numOutputs)
		b.Run(name, func(b *testing.B) {
			f := &TokenTxValidateViewFactory{}
			p := &TokenTxValidateParams{NumInputs: tc.numInputs, NumOutputs: tc.numOutputs}
			input, _ := json.Marshal(p)

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					v, _ := f.NewView(input)
					_, _ = v.Call(nil)
				}
			})
			benchmark.ReportTPS(b)
		})
	}
}
