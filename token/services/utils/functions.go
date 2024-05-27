/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

func IdentityFunc[T any]() func(T) T {
	return func(t T) T { return t }
}
