/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// This file provides generic utility functions including identity functions and nil checking.
// IdentityFunc returns a function that returns its input unchanged.
// IsNil checks if a value is nil using reflection for pointer-like types.
package utils

import "reflect"

func IdentityFunc[T any]() func(T) T {
	return func(t T) T { return t }
}

func IsNil[T any](value T) bool {
	// Use reflection to check if the value is nil
	v := reflect.ValueOf(value)

	return (v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice || v.Kind() == reflect.Map || v.Kind() == reflect.Chan || v.Kind() == reflect.Func || v.Kind() == reflect.Interface) && v.IsNil()
}
