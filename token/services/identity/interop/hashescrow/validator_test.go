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
	script := &he.Script{
		Sender:    sender,
		Recipient: recipient,
		HashInfo: he.HashInfo{
			Hash:         hash,
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

func TestMetadataClaimKeyCheck(t *testing.T) {
	_, script := wrapScript(t, []byte("s"), []byte("r"), []byte("h"))

	preimage := []byte("pre")
	image, err := script.HashInfo.Image(preimage)
	require.NoError(t, err)
	key := he.ClaimKey(image)

	sigRaw, err := json.Marshal(&he.ClaimSignature{
		ClaimantSignature: []byte("sig"),
		Preimage:          preimage,
	})
	require.NoError(t, err)

	act := &actionStub{md: map[string][]byte{key: preimage}}
	got, err := ihe.MetadataClaimKeyCheck(act, script, sigRaw)
	require.NoError(t, err)
	require.Equal(t, key, got)

	act = &actionStub{md: map[string][]byte{}}
	_, err = ihe.MetadataClaimKeyCheck(act, script, sigRaw)
	require.Error(t, err)

	_, err = ihe.MetadataClaimKeyCheck(act, script, []byte("bad-json"))
	require.Error(t, err)
}

func TestMetadataLockKeyCheck(t *testing.T) {
	_, script := wrapScript(t, []byte("s"), []byte("r"), []byte("h"))
	key := he.LockKey(script.HashInfo.Hash)

	act := &actionStub{md: map[string][]byte{key: he.LockValue(script.HashInfo.Hash)}}
	got, err := ihe.MetadataLockKeyCheck(act, script)
	require.NoError(t, err)
	require.Equal(t, key, got)

	act = &actionStub{md: map[string][]byte{}}
	_, err = ihe.MetadataLockKeyCheck(act, script)
	require.Error(t, err)

	act = &actionStub{md: map[string][]byte{key: []byte("bad")}}
	_, err = ihe.MetadataLockKeyCheck(act, script)
	require.Error(t, err)
}
