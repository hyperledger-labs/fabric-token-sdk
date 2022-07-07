/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// G1Array is an array of G1 elements
type G1Array struct {
	Elements []*math.G1
}

// G2Array is an array of G2 elements
type G2Array struct {
	Elements []*math.G2
}

// GTArray is an array of GT elements
type GTArray struct {
	Elements []*math.Gt
}

// Bytes serialize an array of G1 elements
func (a *G1Array) Bytes() ([]byte, error) {
	var raw []byte
	for _, e := range a.Elements {
		if e == nil {
			return nil, errors.Errorf("failed to marshal array of G1")
		}
		bytes := e.Bytes()
		raw = append(raw, bytes...)
	}
	return raw, nil
}

// Bytes serializes an array of G2 elements
func (a *G2Array) Bytes() ([]byte, error) {
	var raw []byte
	for _, e := range a.Elements {
		if e == nil {
			return nil, errors.Errorf("failed to marshal array of G2")
		}
		bytes := e.Bytes()
		raw = append(raw, bytes...)
	}
	return raw, nil
}

// Bytes serializes an array of GT elements
func (a *GTArray) Bytes() ([]byte, error) {
	var raw []byte
	for _, e := range a.Elements {
		if e == nil {
			return nil, errors.Errorf("failed to marshal array of G2")
		}
		bytes := e.Bytes()
		raw = append(raw, bytes...)
	}
	return raw, nil
}

// GetG1Array takes a series of G1 elements and returns the corresponding array
func GetG1Array(elements ...[]*math.G1) *G1Array {
	array := &G1Array{}
	for _, e := range elements {
		array.Elements = append(array.Elements, e...)
	}
	return array
}

// GetG2Array takes a series of G2 elements and returns the corresponding array
func GetG2Array(elements ...[]*math.G2) *G2Array {
	array := &G2Array{}
	for _, e := range elements {
		array.Elements = append(array.Elements, e...)
	}
	return array
}

// GetGTArray takes a series of GT elements and returns the corresponding array
func GetGTArray(elements ...[]*math.Gt) *GTArray {
	array := &GTArray{}
	for _, e := range elements {
		array.Elements = append(array.Elements, e...)
	}
	return array
}

// GetBytesArray takes a series of array of bytes and returns the concatenation
func GetBytesArray(bytes ...[]byte) []byte {
	var array []byte
	for _, b := range bytes {
		array = append(array, b...)
	}
	return array
}

// GetZrArray takes a series of Zr elements and returns the corresponding array
func GetZrArray(elements ...[]*math.Zr) []*math.Zr {
	var array []*math.Zr
	for _, e := range elements {
		array = append(array, e...)
	}
	return array
}

// Sum takes an array of Zr elements and returns the corresponding sum
// modulo the group order c.GroupOrder
func Sum(values []*math.Zr, c *math.Curve) (*math.Zr, error) {
	sum := c.NewZrFromInt(0)
	for i := 0; i < len(values); i++ {
		if values[i] == nil {
			return nil, errors.Errorf("invalid value to be summed")
		}
		sum = c.ModAdd(sum, values[i], c.GroupOrder)
	}
	return sum, nil
}
