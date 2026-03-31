/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package utils provides utility functions for resource cleanup and error handling.
// This file contains helpers for ignoring errors from cleanup operations like Close().
package utils

// IgnoreError runs a function that returns an error and silently ignores the error.
// It is intended to be used when the error from a cleanup operation (e.g., Close)
// is non-critical and can be safely discarded.
//
// Example:
//
//	defer IgnoreError(file.Close)
func IgnoreError(fn func() error) {
	_ = fn()
}

func IgnoreErrorWithOneArg[T any](fn func(t T) error, t T) {
	_ = fn(t)
}
