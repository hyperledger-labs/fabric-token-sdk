/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dedup

import (
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
)

// AndSort returns the elements of source with duplicates removed and the result
// sorted in ascending order.
//
// Lockers call this before acquiring the per-enrollment-ID locks of a request.
// Removing duplicates avoids acquiring (and having to release) the same lock
// twice, and the canonical ascending order is what prevents deadlock: two
// concurrent acquisitions whose enrollment-ID sets intersect always take the
// shared locks in the same order, so they cannot block each other in a cycle.
func AndSort(source []string) []string {
	slice := collections.NewSet(source...).ToSlice()
	slices.Sort(slice)

	return slice
}
