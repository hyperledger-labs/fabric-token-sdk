/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

// IgnoreError runs a function that returns an error and silently ignores the error.
// It is intended to be used when the error from a cleanup operation (e.g., Close)
// is non-critical and can be safely discarded.
//
// Example:
//   defer IgnoreError(file.Close)
func IgnoreError(fn func() error) {
	_ = fn()
}
