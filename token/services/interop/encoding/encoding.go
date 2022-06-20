/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package encoding

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
)

const (
	None Encoding = iota
	Base64
	Hex
	maxEncoding
)

var (
	Encodings = make([]func() EncodingFunc, maxEncoding)
)

type EncodingFunc interface {
	EncodeToString(src []byte) string
}

type Encoding uint

func (e Encoding) EncodingFunc() Encoding {
	return e
}

func (e Encoding) String() string {
	switch e {
	case None:
		return "None"
	case Base64:
		return "Base64"
	case Hex:
		return "Hex"
	default:
		return "unknown Encoding value " + strconv.Itoa(int(e))
	}
}

// New returns a new Encoding.Encoding calculating the given Encoding function. New panics
// if the Encoding function is not linked into the binary.
func (e Encoding) New() EncodingFunc {
	if e < maxEncoding {
		f := Encodings[e]
		if f != nil {
			return f()
		}
	}
	panic("exchange: requested Encoding function #" + strconv.Itoa(int(e)) + " is unavailable")
}

// Available reports whether the given Encoding function is linked into the binary.
func (e Encoding) Available() bool {
	return e < maxEncoding && Encodings[e] != nil
}

// RegisterEncoding registers a function that returns a new instance of the given
// Encoding function. This is intended to be called from the init function in
// packages that implement Encoding functions.
func RegisterEncoding(e Encoding, f func() EncodingFunc) {
	if e >= maxEncoding {
		panic("RegisterEncoding of unknown Encoding function")
	}
	Encodings[e] = f
}

func init() {
	noneEncoding := &noneEncoding{}
	RegisterEncoding(None, func() EncodingFunc {
		return noneEncoding
	})
	RegisterEncoding(Base64, func() EncodingFunc {
		return base64.StdEncoding
	})
	hexEncoding := &hexEncoding{}
	RegisterEncoding(Hex, func() EncodingFunc {
		return hexEncoding
	})
}

type hexEncoding struct{}

func (h hexEncoding) EncodeToString(src []byte) string {
	return hex.EncodeToString(src)
}

type noneEncoding struct{}

func (n noneEncoding) EncodeToString(src []byte) string {
	return string(src)
}
