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
	s.RecipientHashInfo = HashInfo{Hash: []byte("h")}
	require.Error(t, s.Validate())

	s.RecipientHashInfo = HashInfo{Hash: []byte("h"), HashFunc: crypto.SHA256, HashEncoding: encoding.Base64}
	s.SenderHashInfo = HashInfo{Hash: []byte("h2"), HashFunc: crypto.SHA256, HashEncoding: encoding.Base64}
	require.NoError(t, s.Validate())
}

func TestScriptFromBytes(t *testing.T) {
	raw, err := json.Marshal(&Script{
		Sender:    []byte("sender"),
		Recipient: []byte("recipient"),
		RecipientHashInfo: HashInfo{
			Hash:         []byte("h"),
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
		SenderHashInfo: HashInfo{
			Hash:         []byte("h2"),
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
	})
	require.NoError(t, err)

	s := &Script{}
	require.NoError(t, s.FromBytes(raw))
	require.Equal(t, []byte("sender"), []byte(s.Sender))
	require.Equal(t, []byte("recipient"), []byte(s.Recipient))
	require.Equal(t, []byte("h"), s.RecipientHashInfo.Hash)
	require.Equal(t, []byte("h2"), s.SenderHashInfo.Hash)

	require.Error(t, s.FromBytes([]byte("bad-json")))
}

func TestResolveRecipientForPreImage(t *testing.T) {
	s := &Script{
		Sender:    []byte("sender"),
		Recipient: []byte("recipient"),
		RecipientHashInfo: HashInfo{
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
		SenderHashInfo: HashInfo{
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
	}

	recipientPreImage := []byte("recipient-secret")
	recipientImage, err := s.RecipientHashInfo.Image(recipientPreImage)
	require.NoError(t, err)
	s.RecipientHashInfo.Hash = recipientImage

	senderPreImage := []byte("sender-secret")
	senderImage, err := s.SenderHashInfo.Image(senderPreImage)
	require.NoError(t, err)
	s.SenderHashInfo.Hash = senderImage

	owner, image, err := s.ResolveRecipientForPreImage(recipientPreImage)
	require.NoError(t, err)
	require.Equal(t, []byte("recipient"), []byte(owner))
	require.Equal(t, recipientImage, image)

	owner, image, err = s.ResolveRecipientForPreImage(senderPreImage)
	require.NoError(t, err)
	require.Equal(t, []byte("sender"), []byte(owner))
	require.Equal(t, senderImage, image)

	_, _, err = s.ResolveRecipientForPreImage([]byte("wrong"))
	require.Error(t, err)
}

func TestKeys(t *testing.T) {
	recipientHash := []byte{0x01, 0x02}
	senderHash := []byte{0x03, 0x04}

	k := ClaimKey(recipientHash, senderHash)
	require.Contains(t, k, "hashescrow.cm:")

	k = LockKey(recipientHash, senderHash)
	require.Contains(t, k, "hashescrow.lh:")

	v, err := LockValue(recipientHash, senderHash)
	require.NoError(t, err)
	require.NotEmpty(t, v)
}
