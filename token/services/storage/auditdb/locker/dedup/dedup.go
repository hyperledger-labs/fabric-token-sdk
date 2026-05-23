/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dedup

import (
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
)

func AndSort(source []string) []string {
	slice := collections.NewSet(source...).ToSlice()
	slices.Sort(slice)

	return slice
}
