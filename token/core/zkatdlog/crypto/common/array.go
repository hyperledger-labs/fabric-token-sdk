/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
)

type G1Array struct {
	Elements []*bn256.G1
}

type G2Array struct {
	Elements []*bn256.G2
}

type GTArray struct {
	Elements []*bn256.GT
}

func (a *G1Array) Bytes() []byte {
	var raw []byte
	for _, e := range a.Elements {
		bytes := e.Bytes()
		raw = append(raw, bytes...)
	}
	return raw
}

func (a *G2Array) Bytes() []byte {
	var raw []byte
	for _, e := range a.Elements {
		bytes := e.Bytes()
		raw = append(raw, bytes...)
	}
	return raw
}

func (a *GTArray) Bytes() []byte {
	var raw []byte
	for _, e := range a.Elements {
		bytes := e.Bytes()
		raw = append(raw, bytes...)
	}
	return raw
}

func GetG1Array(elements ...[]*bn256.G1) *G1Array {
	array := &G1Array{}
	for _, e := range elements {
		array.Elements = append(array.Elements, e...)
	}
	return array
}

func GetG2Array(elements ...[]*bn256.G2) *G2Array {
	array := &G2Array{}
	for _, e := range elements {
		array.Elements = append(array.Elements, e...)
	}
	return array
}

func GetGTArray(elements ...[]*bn256.GT) *GTArray {
	array := &GTArray{}
	for _, e := range elements {
		array.Elements = append(array.Elements, e...)
	}
	return array
}

func GetBytesArray(bytes ...[]byte) []byte {
	var array []byte
	for _, b := range bytes {
		array = append(array, b...)
	}
	return array
}

func GetZrArray(elements ...[]*bn256.Zr) []*bn256.Zr {
	var array []*bn256.Zr
	for _, e := range elements {
		array = append(array, e...)
	}
	return array
}
