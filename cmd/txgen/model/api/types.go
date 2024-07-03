/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

type Amount = int64
type UUID [16]byte

func (u UUID) String() string {
	return string(u[:])
}
