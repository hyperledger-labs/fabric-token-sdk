/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"crypto"
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/stretchr/testify/require"
)

func TestScriptValidate(t *testing.T) {
	s := &Script{}
	require.EqualError(t, s.Validate(), "sender not set")

	s.Sender = []byte("sender")
	require.EqualError(t, s.Validate(), "recipient not set")

	s.Recipient = []byte("recipient")
	s.HashInfo = HashInfo{Hash: []byte("h")}
	require.Error(t, s.Validate())

	s.HashInfo = HashInfo{Hash: []byte("h"), HashFunc: crypto.SHA256, HashEncoding: encoding.Base64}
	require.NoError(t, s.Validate())
}

func TestScriptFromBytes(t *testing.T) {
	raw, err := json.Marshal(&Script{
		Sender:    []byte("sender"),
		Recipient: []byte("recipient"),
		HashInfo: HashInfo{
			Hash:         []byte("h"),
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
	})
	require.NoError(t, err)

	s := &Script{}
	require.NoError(t, s.FromBytes(raw))
	require.Equal(t, []byte("sender"), []byte(s.Sender))
	require.Equal(t, []byte("recipient"), []byte(s.Recipient))
	require.Equal(t, []byte("h"), s.HashInfo.Hash)

	require.Error(t, s.FromBytes([]byte("bad-json")))
}

func TestKeys(t *testing.T) {
	k := ClaimKey([]byte{0x01, 0x02})
	require.Equal(t, "hashescrow.cpi0102", k)

	k = LockKey([]byte{0x0a})
	require.Equal(t, "hashescrow.lh0a", k)

	v := LockValue([]byte{0x0a, 0x0b})
	require.Equal(t, []byte("0a0b"), v)
}
