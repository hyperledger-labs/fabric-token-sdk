/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storage

// Iterator defines a generic iterator
type Iterator[k any] interface {
	HasNext() bool
	Close() error
	Next() (k, error)
}
