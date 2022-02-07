/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// RWSet models a transaction's read-write set
type RWSet interface {
	// Done signals the end of the manipulation of this read-write set
	Done()
}
