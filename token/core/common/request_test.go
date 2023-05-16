/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestSerialization(t *testing.T) {
	r := &TokenRequest{
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

	r2 := &TokenRequest{}
	err = r2.FromBytes(raw)
	assert.NoError(t, err)
	raw2, err := r2.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, raw, raw2)

	mRaw, err := r.MarshalToAudit("id", nil)
	assert.NoError(t, err)
	mRaw2, err := r2.MarshalToAudit("id", nil)
	assert.NoError(t, err)

	assert.Equal(t, mRaw, mRaw2)
}
