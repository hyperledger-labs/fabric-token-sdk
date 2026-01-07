/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Separator is used to delimit to end an array of bytes.
// The bytes are the bytes of hex-encoded string.
const Separator = "||"

// G1Array is an array of G1 elements
type G1Array []*math.G1

// Bytes serialize an array of G1 elements
func (a G1Array) Bytes() ([]byte, error) {
	raw := make([][]byte, len([]*math.G1(a)))
	for i, e := range []*math.G1(a) {
		if e == nil {
			return nil, errors.Errorf("failed to marshal array of G1")
		}
		raw[i] = e.Bytes()
	}
	// join the serialization of the group elements with the predefined separator.
	return bytes.Join(raw, []byte(Separator)), nil
}

// GetG1Array takes a series of G1 elements and returns the corresponding array
func GetG1Array(elements ...[]*math.G1) G1Array {
	// compute length
	length := 0
	for _, e := range elements {
		length += len(e)
	}
	s := make([]*math.G1, 0, length)
	for _, e := range elements {
		s = append(s, e...)
	}
	return s
}
