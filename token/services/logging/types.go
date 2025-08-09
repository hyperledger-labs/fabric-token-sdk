/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
)

func Prefix(id string) fmt.Stringer {
	return logging.Prefix(id)
}

func Printable(id string) fmt.Stringer {
	return logging.Printable(id)
}

func Keys[K comparable, V any](m map[K]V) fmt.Stringer {
	return logging.Keys[K, V](m)
}

func Identifier(f any) fmt.Stringer { return logging.Identifier(f) }

func Base64(b []byte) fmt.Stringer {
	return logging.Base64(b)
}
