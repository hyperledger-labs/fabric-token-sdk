/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func TestRequestSerialization(t *testing.T) {
	r := NewRequest(nil, "hello world")
	r.Actions = &driver.TokenRequest{
		Issues: [][]byte{
			[]byte("issue1"),
			[]byte("issue2"),
		},
		Transfers:         [][]byte{[]byte("transfer1")},
		Signatures:        [][]byte{[]byte("signature1")},
		AuditorSignatures: [][]byte{[]byte("auditor_signature1")},
	}
	raw, err := r.Bytes()
	assert.NoError(t, err)

	r2 := NewRequest(nil, "")
	err = r2.FromBytes(raw)
	assert.NoError(t, err)
	raw2, err := r2.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, raw, raw2)

	mRaw, err := r.MarshallToAudit()
	assert.NoError(t, err)
	mRaw2, err := r2.MarshallToAudit()
	assert.NoError(t, err)

	assert.Equal(t, mRaw, mRaw2)
}
