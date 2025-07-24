/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package slices

func GetUnique[V any](v []V) V {
	return v[0]
}

func GetAny[V any](v []V) V {
	return v[0]
}

func GetFirst[V any](v []V) V {
	return v[0]
}

func GenericSliceOfPointers[T any](size int) []*T {
	slice := make([]*T, size)
	for i := range slice {
		var zero T
		slice[i] = &zero
	}
	return slice
}
