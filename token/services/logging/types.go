/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
)

func Prefix(id string) fmt.Stringer {
	return prefix(id)
}

type prefix string

func (w prefix) String() string {
	s := string(w)
	if len(s) <= 20 {
		return strings.ToValidUTF8(s, "X")
	}
	return fmt.Sprintf("%s~%s", strings.ToValidUTF8(s[:20], "X"), hash.Hashable(s).String())
}

func Printable(id string) fmt.Stringer {
	return printable(id)
}

type printable string

func (w printable) String() string {
	s := string(w)
	return strings.ToValidUTF8(s, "X")
}

func Keys[K comparable, V any](m map[K]V) fmt.Stringer {
	return logging.Keys[K, V](m)
}

func Identifier(f any) fmt.Stringer { return logging.Identifier(f) }

func Base64(b []byte) fmt.Stringer {
	return logging.Base64(b)
}
