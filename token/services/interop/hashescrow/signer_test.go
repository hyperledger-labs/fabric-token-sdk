/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow_test

import (
	"crypto"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/stretchr/testify/require"
)

func validClaimSigBytes(t *testing.T, preimage, claimantSig []byte) []byte {
	t.Helper()
	raw, err := json.Marshal(hashescrow.ClaimSignature{Preimage: preimage, ClaimantSignature: claimantSig})
	require.NoError(t, err)

	return raw
}

func claimHashInfo(preimage []byte) hashescrow.HashInfo {
	h := crypto.SHA256.New()
	_, _ = h.Write(preimage)

	return hashescrow.HashInfo{
		Hash:         []byte(encoding.Base64.New().EncodeToString(h.Sum(nil))),
		HashFunc:     crypto.SHA256,
		HashEncoding: encoding.Base64,
	}
}

func TestClaimSignerSign(t *testing.T) {
	preimage := []byte("secret")
	msg := []byte("txrequest")
	expectedSig := []byte("claimant-sig")

	signer := &mock.Signer{}
	signer.SignReturns(expectedSig, nil)

	cs := &hashescrow.ClaimSigner{Claimant: signer, Preimage: preimage}
	raw, err := cs.Sign(msg)
	require.NoError(t, err)

	var claimSig hashescrow.ClaimSignature
	require.NoError(t, json.Unmarshal(raw, &claimSig))
	require.Equal(t, preimage, claimSig.Preimage)
	require.Equal(t, expectedSig, claimSig.ClaimantSignature)
	require.Equal(t, 1, signer.SignCallCount())
	require.Equal(t, append(msg, preimage...), signer.SignArgsForCall(0))
}

func TestClaimSignerSignError(t *testing.T) {
	signer := &mock.Signer{}
	signer.SignReturns(nil, errors.New("sign failed"))

	cs := &hashescrow.ClaimSigner{Claimant: signer, Preimage: []byte("p")}
	_, err := cs.Sign([]byte("req"))
	require.EqualError(t, err, "sign failed")
}

func TestClaimVerifierVerify(t *testing.T) {
	preimage := []byte("secret")
	msg := []byte("txrequest")

	verifier := &mock.Verifier{}
	verifier.VerifyReturns(nil)

	cv := &hashescrow.ClaimVerifier{Claimant: verifier, HashInfo: claimHashInfo(preimage)}
	require.NoError(t, cv.Verify(msg, validClaimSigBytes(t, preimage, []byte("sig"))))
	vmsg, _ := verifier.VerifyArgsForCall(0)
	require.Equal(t, append(msg, preimage...), vmsg)
}

func TestClaimVerifierErrors(t *testing.T) {
	cv := &hashescrow.ClaimVerifier{Claimant: &mock.Verifier{}, HashInfo: claimHashInfo([]byte("p"))}
	require.Error(t, cv.Verify([]byte("req"), []byte("bad-json")))

	verifier := &mock.Verifier{}
	verifier.VerifyReturns(errors.New("bad sig"))
	cv = &hashescrow.ClaimVerifier{Claimant: verifier, HashInfo: claimHashInfo([]byte("p"))}
	err := cv.Verify([]byte("req"), validClaimSigBytes(t, []byte("p"), []byte("sig")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to verify claimant signature")

	verifier = &mock.Verifier{}
	verifier.VerifyReturns(nil)
	cv = &hashescrow.ClaimVerifier{
		Claimant: verifier,
		HashInfo: claimHashInfo([]byte("different-preimage")),
	}
	err = cv.Verify([]byte("req"), validClaimSigBytes(t, []byte("p"), []byte("sig")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}

func TestVerifierVerify(t *testing.T) {
	preimage := []byte("secret")
	msg := []byte("txrequest")
	sigma := validClaimSigBytes(t, preimage, []byte("sig"))

	recipientVerifier := &mock.Verifier{}
	recipientVerifier.VerifyReturns(nil)
	senderVerifier := &mock.Verifier{}
	v := &hashescrow.Verifier{
		Recipient: recipientVerifier,
		Sender:    senderVerifier,
		HashInfo:  claimHashInfo(preimage),
	}
	require.NoError(t, v.Verify(msg, sigma))
	require.Equal(t, 1, recipientVerifier.VerifyCallCount())
	require.Equal(t, 0, senderVerifier.VerifyCallCount())

	recipientVerifier = &mock.Verifier{}
	recipientVerifier.VerifyReturns(errors.New("bad recipient sig"))
	senderVerifier = &mock.Verifier{}
	senderVerifier.VerifyReturns(nil)
	v = &hashescrow.Verifier{
		Recipient: recipientVerifier,
		Sender:    senderVerifier,
		HashInfo:  claimHashInfo(preimage),
	}
	require.NoError(t, v.Verify(msg, sigma))
	require.Equal(t, 1, senderVerifier.VerifyCallCount())

	recipientVerifier = &mock.Verifier{}
	recipientVerifier.VerifyReturns(errors.New("bad recipient sig"))
	senderVerifier = &mock.Verifier{}
	senderVerifier.VerifyReturns(errors.New("bad sender sig"))
	v = &hashescrow.Verifier{
		Recipient: recipientVerifier,
		Sender:    senderVerifier,
		HashInfo:  claimHashInfo(preimage),
	}
	err := v.Verify(msg, sigma)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed verifying hash escrow claim signature")
}
