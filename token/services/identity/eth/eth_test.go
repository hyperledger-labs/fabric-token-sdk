/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package eth_test

import (
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/eth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateKey is a test helper that creates a fresh secp256k1 key pair.
func generateKey(t *testing.T) (*secp256k1.PrivateKey, *secp256k1.PublicKey) {
	t.Helper()
	priv, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)

	return priv, priv.PubKey()
}

// ---------------------------------------------------------------------------
// Signer / Verifier round-trip tests
// ---------------------------------------------------------------------------

func TestSignVerify_RoundTrip(t *testing.T) {
	priv, pub := generateKey(t)
	signer := eth.NewSigner(priv)
	verifier := eth.NewVerifier(pub)

	message := []byte("approve token transfer tx-001")
	sig, err := signer.Sign(message)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	require.NoError(t, verifier.Verify(message, sig))
}

func TestVerify_WrongMessage(t *testing.T) {
	priv, pub := generateKey(t)
	signer := eth.NewSigner(priv)
	verifier := eth.NewVerifier(pub)

	sig, err := signer.Sign([]byte("original message"))
	require.NoError(t, err)

	err = verifier.Verify([]byte("tampered message"), sig)
	require.Error(t, err)
}

func TestVerify_WrongKey(t *testing.T) {
	priv, _ := generateKey(t)
	_, differentPub := generateKey(t)

	signer := eth.NewSigner(priv)
	verifier := eth.NewVerifier(differentPub)

	sig, err := signer.Sign([]byte("hello"))
	require.NoError(t, err)

	err = verifier.Verify([]byte("hello"), sig)
	require.Error(t, err)
}

func TestSign_NilKey_ReturnsError(t *testing.T) {
	signer := eth.NewSigner(nil)
	_, err := signer.Sign([]byte("msg"))
	require.Error(t, err)
}

func TestVerify_NilKey_ReturnsError(t *testing.T) {
	verifier := eth.NewVerifier(nil)
	err := verifier.Verify([]byte("msg"), []byte("sig"))
	require.Error(t, err)
}

func TestVerify_MalformedSignature(t *testing.T) {
	_, pub := generateKey(t)
	verifier := eth.NewVerifier(pub)
	err := verifier.Verify([]byte("msg"), []byte("not-a-der-signature"))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// AddressFromPublicKey tests
// ---------------------------------------------------------------------------

func TestAddressFromPublicKey_Deterministic(t *testing.T) {
	_, pub := generateKey(t)
	addr1 := eth.AddressFromPublicKey(pub)
	addr2 := eth.AddressFromPublicKey(pub)
	assert.Equal(t, addr1, addr2)
}

func TestAddressFromPublicKey_DifferentKeys_DifferentAddresses(t *testing.T) {
	_, pub1 := generateKey(t)
	_, pub2 := generateKey(t)
	addr1 := eth.AddressFromPublicKey(pub1)
	addr2 := eth.AddressFromPublicKey(pub2)
	assert.NotEqual(t, addr1, addr2)
}

func TestAddressFromPublicKey_Length(t *testing.T) {
	_, pub := generateKey(t)
	addr := eth.AddressFromPublicKey(pub)
	assert.Len(t, addr, 20)
}

// ---------------------------------------------------------------------------
// EIP-712 HashEndorsementRequest tests
// ---------------------------------------------------------------------------

var testDomain = eth.Domain{
	Name:    "FabricTokenSDK",
	Version: "1",
	ChainID: 1,
}

func TestHashEndorsementRequest_Deterministic(t *testing.T) {
	req := eth.EndorsementRequest{
		TMSID:    "testnet:ch1:ns1",
		TxID:     "tx-abc-123",
		Deadline: 9999999999,
	}

	h1 := eth.HashEndorsementRequest(testDomain, req)
	h2 := eth.HashEndorsementRequest(testDomain, req)
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 32)
}

func TestHashEndorsementRequest_DifferentTxIDs_DifferentHashes(t *testing.T) {
	req1 := eth.EndorsementRequest{TMSID: "net:ch:ns", TxID: "tx-1", Deadline: 0}
	req2 := eth.EndorsementRequest{TMSID: "net:ch:ns", TxID: "tx-2", Deadline: 0}

	h1 := eth.HashEndorsementRequest(testDomain, req1)
	h2 := eth.HashEndorsementRequest(testDomain, req2)
	assert.NotEqual(t, h1, h2)
}

func TestHashEndorsementRequest_DifferentDomains_DifferentHashes(t *testing.T) {
	req := eth.EndorsementRequest{TMSID: "net:ch:ns", TxID: "tx-1", Deadline: 0}

	domainA := eth.Domain{Name: "SDKv1", Version: "1", ChainID: 1}
	domainB := eth.Domain{Name: "SDKv1", Version: "1", ChainID: 137} // Polygon

	h1 := eth.HashEndorsementRequest(domainA, req)
	h2 := eth.HashEndorsementRequest(domainB, req)
	assert.NotEqual(t, h1, h2)
}

func TestHashEndorsementRequest_DifferentDeadlines_DifferentHashes(t *testing.T) {
	req1 := eth.EndorsementRequest{TMSID: "net:ch:ns", TxID: "tx-1", Deadline: 0}
	req2 := eth.EndorsementRequest{TMSID: "net:ch:ns", TxID: "tx-1", Deadline: 1700000000}

	h1 := eth.HashEndorsementRequest(testDomain, req1)
	h2 := eth.HashEndorsementRequest(testDomain, req2)
	assert.NotEqual(t, h1, h2)
}

// ---------------------------------------------------------------------------
// End-to-end: sign an EIP-712 endorsement and verify it
// ---------------------------------------------------------------------------

func TestEndorseAndVerify_EIP712(t *testing.T) {
	priv, pub := generateKey(t)
	signer := eth.NewSigner(priv)
	verifier := eth.NewVerifier(pub)

	req := eth.EndorsementRequest{
		TMSID:    "testnet:mychannel:token-ns",
		TxID:     "transfer-tx-xyz",
		Deadline: 1800000000,
	}

	// The endorser hashes the request with EIP-712 and signs.
	digest := eth.HashEndorsementRequest(testDomain, req)
	sig, err := signer.Sign(digest)
	require.NoError(t, err)

	// The verifier independently re-derives the digest and confirms the signature.
	require.NoError(t, verifier.Verify(digest, sig))
}
