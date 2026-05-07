/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
)

// ClaimSignature carries the preimage and the claimant signature.
type ClaimSignature struct {
	ClaimantSignature []byte
	Preimage          []byte
}

// ClaimSigner signs token request bytes and embeds the preimage.
type ClaimSigner struct {
	Claimant driver.Signer
	Preimage []byte
}

// Sign returns a signature of the claimant over token request and preimage.
func (cs *ClaimSigner) Sign(tokenRequestAndTxID []byte) ([]byte, error) {
	msg := concatTokenRequestTxIDPreimage(tokenRequestAndTxID, cs.Preimage)
	sigma, err := cs.Claimant.Sign(msg)
	if err != nil {
		return nil, err
	}

	claimSignature := ClaimSignature{
		Preimage:          cs.Preimage,
		ClaimantSignature: sigma,
	}

	return json.Marshal(claimSignature)
}

func concatTokenRequestTxIDPreimage(tokenRequestAndTxID []byte, preImage []byte) []byte {
	var msg []byte
	msg = append(msg, tokenRequestAndTxID...)
	msg = append(msg, preImage...)

	return msg
}

// ClaimVerifier checks preimage validity and claimant signature.
type ClaimVerifier struct {
	Claimant driver.Verifier
	HashInfo HashInfo
}

func (cv *ClaimVerifier) Verify(tokenRequestAndTxID, claimSignature []byte) error {
	sig := &ClaimSignature{}
	err := json.Unmarshal(claimSignature, sig)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal claim signature")
	}

	msg := concatTokenRequestTxIDPreimage(tokenRequestAndTxID, sig.Preimage)
	if err := cv.Claimant.Verify(msg, sig.ClaimantSignature); err != nil {
		return errors.WithMessagef(err, "failed to verify claimant signature")
	}

	image, err := (&htlc.HashInfo{
		Hash:         cv.HashInfo.Hash,
		HashFunc:     cv.HashInfo.HashFunc,
		HashEncoding: cv.HashInfo.HashEncoding,
	}).Image(sig.Preimage)
	if err != nil {
		return err
	}
	if err := (&htlc.HashInfo{Hash: cv.HashInfo.Hash}).Compare(image); err != nil {
		return errors.Errorf("hash mismatch: %s", err)
	}

	return nil
}

// Verifier validates claims for hash-based escrow scripts.
// A valid claim is signed by either recipient or sender and carries a valid preimage.
type Verifier struct {
	Recipient driver.Verifier
	Sender    driver.Verifier
	HashInfo  HashInfo
}

func (v *Verifier) Verify(msg []byte, sigma []byte) error {
	recipientClaimVerifier := &ClaimVerifier{
		Claimant: v.Recipient,
		HashInfo: v.HashInfo,
	}
	if err := recipientClaimVerifier.Verify(msg, sigma); err == nil {
		return nil
	}

	senderClaimVerifier := &ClaimVerifier{
		Claimant: v.Sender,
		HashInfo: v.HashInfo,
	}
	if err := senderClaimVerifier.Verify(msg, sigma); err != nil {
		return errors.WithMessagef(err, "failed verifying hash escrow claim signature")
	}

	return nil
}
