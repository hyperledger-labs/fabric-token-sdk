/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow_test

import (
	"crypto"
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	ihe "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/hashescrow"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	he "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/stretchr/testify/require"
)

type actionStub struct {
	md map[string][]byte
}

func (a *actionStub) GetMetadata() map[string][]byte {
	return a.md
}

func wrapScript(t *testing.T, sender, recipient []byte, hash []byte) ([]byte, *he.Script) {
	t.Helper()
	recipientHash := append([]byte{}, hash...)
	senderHash := append([]byte{}, hash...)
	senderHash = append(senderHash, byte('2'))
	script := &he.Script{
		Sender:    sender,
		Recipient: recipient,
		RecipientHashInfo: he.HashInfo{
			Hash:         recipientHash,
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
		SenderHashInfo: he.HashInfo{
			Hash:         senderHash,
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
	}
	rawScript, err := json.Marshal(script)
	require.NoError(t, err)
	rawOwner, err := identity.WrapWithType(he.ScriptType, rawScript)
	require.NoError(t, err)

	return rawOwner, script
}

func TestVerifyOwner(t *testing.T) {
	rawOwner, _ := wrapScript(t, []byte("s"), []byte("r"), []byte("h"))

	script, err := ihe.VerifyOwner(rawOwner, []byte("r"))
	require.NoError(t, err)
	require.Equal(t, identity.Identity("s"), script.Sender)
	require.Equal(t, identity.Identity("r"), script.Recipient)

	script, err = ihe.VerifyOwner(rawOwner, []byte("s"))
	require.NoError(t, err)
	require.Equal(t, identity.Identity("s"), script.Sender)

	_, err = ihe.VerifyOwner(rawOwner, []byte("x"))
	require.Error(t, err)

	_, err = ihe.VerifyOwner([]byte("bad"), []byte("r"))
	require.Error(t, err)

	typed := identity.TypedIdentity{Type: identity.Type(99), Identity: []byte("x")}
	b, err := typed.Bytes()
	require.NoError(t, err)
	_, err = ihe.VerifyOwner(b, []byte("r"))
	require.Error(t, err)
}

func TestClaimMetadataCheck(t *testing.T) {
	_, script := wrapScript(t, []byte("s"), []byte("r"), []byte("h"))

	preimage := []byte("pre")
	resolvedHash, err := script.RecipientHashInfo.Image(preimage)
	require.NoError(t, err)
	script.RecipientHashInfo.Hash = resolvedHash
	key := he.ClaimKey(script.RecipientHashInfo.Hash, script.SenderHashInfo.Hash)
	value, err := he.ClaimValue(preimage, "recipient")
	require.NoError(t, err)

	sigRaw, err := json.Marshal(&he.ClaimSignature{
		Preimage: preimage,
	})
	require.NoError(t, err)
	claim, err := ihe.ClaimFromSignature(sigRaw)
	require.NoError(t, err)
	resolvedOwner, _, claimedBy, err := ihe.ResolveOwnerAndHash(script, claim.Preimage)
	require.NoError(t, err)
	require.Equal(t, identity.Identity("r"), identity.Identity(resolvedOwner))
	require.Equal(t, "recipient", claimedBy)

	act := &actionStub{md: map[string][]byte{key: value}}
	got, err := ihe.ClaimMetadataCheck(act, script, claim.Preimage, claimedBy)
	require.NoError(t, err)
	require.Equal(t, key, got)

	act = &actionStub{md: map[string][]byte{}}
	_, err = ihe.ClaimMetadataCheck(act, script, claim.Preimage, claimedBy)
	require.Error(t, err)

	_, err = ihe.ClaimFromSignature([]byte("bad-json"))
	require.Error(t, err)
}

func TestLockMetadataCheck(t *testing.T) {
	_, script := wrapScript(t, []byte("s"), []byte("r"), []byte("h"))
	key := he.LockKey(script.RecipientHashInfo.Hash, script.SenderHashInfo.Hash)
	lockValue, err := he.LockValue(script.RecipientHashInfo.Hash, script.SenderHashInfo.Hash)
	require.NoError(t, err)

	act := &actionStub{md: map[string][]byte{
		key: lockValue,
	}}
	got, err := ihe.LockMetadataCheck(act, script)
	require.NoError(t, err)
	require.Equal(t, key, got)

	act = &actionStub{md: map[string][]byte{}}
	_, err = ihe.LockMetadataCheck(act, script)
	require.Error(t, err)

	act = &actionStub{md: map[string][]byte{
		key: []byte("bad"),
	}}
	_, err = ihe.LockMetadataCheck(act, script)
	require.Error(t, err)
}
