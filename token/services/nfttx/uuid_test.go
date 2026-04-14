/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"crypto/rand"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type badReader struct{}

func (b *badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("bad read")
}

func TestUUIDGenerateBytes(t *testing.T) {
	uuid := GenerateBytesUUID()
	assert.Len(t, uuid, 16)
	// variant bits; see section 4.1.1
	assert.Equal(t, byte(0x80), uuid[8]&0xc0)
	// version 4 (pseudo-random); see section 4.1.3
	assert.Equal(t, byte(0x40), uuid[6]&0xf0)
}

func TestUUIDGenerateBytesPanic(t *testing.T) {
	oldReader := rand.Reader
	defer func() { rand.Reader = oldReader }()
	rand.Reader = &badReader{}

	assert.Panics(t, func() {
		GenerateBytesUUID()
	})
}

func TestUUIDGenerateUUID(t *testing.T) {
	u := GenerateUUID()
	assert.Len(t, u, 36)
	assert.Equal(t, uint8('-'), u[8])
	assert.Equal(t, uint8('-'), u[13])
	assert.Equal(t, uint8('-'), u[18])
	assert.Equal(t, uint8('-'), u[23])
}

func TestIDBytesToStr(t *testing.T) {
	b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	s := idBytesToStr(b)
	assert.Equal(t, "00010203-0405-0607-0809-0a0b0c0d0e0f", s)
}
