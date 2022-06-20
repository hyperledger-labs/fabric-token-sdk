/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package encoding_test

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/stretchr/testify/assert"
)

func TestEncodingNone(t *testing.T) {
	msg := []byte("hello world")
	e := encoding.None.New()
	o1 := e.EncodeToString(msg)
	assert.Equal(t, o1, string(msg))
}

func TestEncodingBase64(t *testing.T) {
	msg := []byte("hello world")
	e := encoding.Base64.New()
	o1 := e.EncodeToString(msg)
	o2 := base64.StdEncoding.EncodeToString(msg)
	assert.Equal(t, o1, o2)
}

func TestEncodingHex(t *testing.T) {
	msg := []byte("hello world")
	e := encoding.Hex.New()
	o1 := e.EncodeToString(msg)
	o2 := hex.EncodeToString(msg)
	assert.Equal(t, o1, o2)
}
