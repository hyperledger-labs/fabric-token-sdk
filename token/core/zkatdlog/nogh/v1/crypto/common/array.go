/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto"
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
		st := hex.EncodeToString(e.Bytes())
		raw[i] = []byte(st)
	}
	clear(b)
	return crypto.AppendFixed32(b, raw), nil
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
