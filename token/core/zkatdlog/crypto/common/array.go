/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"encoding/hex"

	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// Separator is used to delimit to end an array of bytes.
// The bytes are the bytes of hex-encoded string.
const Separator = "||"

// G1Array is an array of G1 elements
type G1Array []*math.G1

// Bytes serialize an array of G1 elements
func (a *G1Array) Bytes() ([]byte, error) {

	var raw []byte
	for _, e := range []*math.G1(*a) {
		if e == nil {
			return nil, errors.Errorf("failed to marshal array of G1")
		}
		st := hex.EncodeToString(e.Bytes())
		raw = append(raw, []byte(Separator)...)
		raw = append(raw, []byte(st)...)

	}
	return raw, nil
}

// GetG1Array takes a series of G1 elements and returns the corresponding array
func GetG1Array(elements ...[]*math.G1) *G1Array {
	var array []*math.G1
	for _, e := range elements {
		array = append(array, e...)
	}
	a := G1Array(array)
	return &a
}
