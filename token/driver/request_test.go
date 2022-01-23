/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenRequestSerialization(t *testing.T) {
	req := &TokenRequest{
		Issues: [][]byte{
			[]byte("issue1"),
			[]byte("issue2"),
		},
		Transfers:         [][]byte{[]byte("transfer1")},
		Signatures:        [][]byte{[]byte("signature1")},
		AuditorSignatures: [][]byte{[]byte("auditor_signature1")},
	}
	raw, err := req.Bytes()
	assert.NoError(t, err)

	req2 := &TokenRequest{}
	err = req2.FromBytes(raw)
	assert.NoError(t, err)
	assert.Equal(t, req, req2)
}
