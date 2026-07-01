/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"testing"

	"github.com/LFDT-Panurus/panurus/token"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withdrawalProofArgs builds the proof the same way withdrawal.go
// does on both sides (walletID/multiSig/policy are fixed for withdrawal).
func withdrawalProofArgs(tmsID token.TMSID, identity view.Identity, nonce []byte, sessionID, contextID string) ([]byte, error) {
	return buildAttestationMessage(tmsID, nil, identity, false, "", nonce, sessionID, contextID)
}

// Requester and issuer must build the same bytes from the same inputs, or the
// signature would never verify.
func TestWithdrawalProof_BothSidesAgree(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	identity := view.Identity("alice")
	nonce := []byte{0x01, 0x02, 0x03}

	requesterSide, err := withdrawalProofArgs(tmsID, identity, nonce, "session-1", "context-1")
	require.NoError(t, err)
	issuerSide, err := withdrawalProofArgs(tmsID, identity, nonce, "session-1", "context-1")
	require.NoError(t, err)

	assert.Equal(t, requesterSide, issuerSide, "requester and issuer must reconstruct identical proof bytes")
}

// Changing any signed field must change the bytes, so a signature can't be
// replayed onto a different identity, nonce, session or context.
func TestWithdrawalProof_Binds(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	base, err := withdrawalProofArgs(tmsID, view.Identity("alice"), []byte{0xAA}, "s1", "c1")
	require.NoError(t, err)

	cases := map[string][]byte{
		"different identity": mustProof(t, tmsID, view.Identity("bob"), []byte{0xAA}, "s1", "c1"),
		"different nonce":    mustProof(t, tmsID, view.Identity("alice"), []byte{0xBB}, "s1", "c1"),
		"different session":  mustProof(t, tmsID, view.Identity("alice"), []byte{0xAA}, "s2", "c1"),
		"different context":  mustProof(t, tmsID, view.Identity("alice"), []byte{0xAA}, "s1", "c2"),
		"different namespace": mustProof(t, token.TMSID{Network: "net", Channel: "ch", Namespace: "other"},
			view.Identity("alice"), []byte{0xAA}, "s1", "c1"),
	}
	for name, other := range cases {
		assert.NotEqualf(t, base, other, "proof must change when %s changes", name)
	}
}

// "AB"+"CD" and "ABC"+"D" would collide if fields were just concatenated. The
// framed message must keep them distinct so bytes can't shift between fields.
func TestWithdrawalProof_FieldBoundary(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	a := mustProof(t, tmsID, view.Identity("AB"), []byte("CD"), "s", "c")
	b := mustProof(t, tmsID, view.Identity("ABC"), []byte("D"), "s", "c")
	assert.NotEqual(t, a, b, "field boundary between identity and nonce must be unambiguous")
}

func mustProof(t *testing.T, tmsID token.TMSID, identity view.Identity, nonce []byte, sessionID, contextID string) []byte {
	t.Helper()
	msg, err := withdrawalProofArgs(tmsID, identity, nonce, sessionID, contextID)
	require.NoError(t, err)

	return msg
}
