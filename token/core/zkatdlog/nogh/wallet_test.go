/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"strings"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
)

var (
	prefix = "testnetwork.testchannel.testnamespace.default"
	id     = hash.Hashable("hello world").RawString()
)

func BenchmarkConcatenation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = prefix + id
	}
}

func BenchmarkBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.WriteString(prefix)
		sb.WriteString(id)
		_ = sb.String()
	}
}
