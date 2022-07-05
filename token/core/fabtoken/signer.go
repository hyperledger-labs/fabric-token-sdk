/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

// ClaimSignature encodes the exchange claim signature
type ClaimSignature struct {
	RecipientSignature []byte
	Preimage           []byte
}

// ClaimSigner is the signer for an exchange claim
type ClaimSigner struct {
	Recipient driver.Signer
	Preimage  []byte
}

func (cs *ClaimSigner) Sign(tokenRequestAndTxID []byte) ([]byte, error) {
	//tokenRequest []byte, txID string
	msg := concatTokenRequestTxIDPreimage(tokenRequestAndTxID, cs.Preimage)
	sigma, err := cs.Recipient.Sign(msg)
	if err != nil {
		return nil, err
	}

	claimSignature := ClaimSignature{
		Preimage:           cs.Preimage,
		RecipientSignature: sigma,
	}
	return json.Marshal(claimSignature)
}

func concatTokenRequestTxIDPreimage(tokenRequestAndTxID []byte, preImage []byte) []byte {
	var msg []byte
	msg = append(msg, tokenRequestAndTxID...)
	msg = append(msg, preImage...)
	return msg
}

// ClaimVerifier is the verifier of an exchange claim
type ClaimVerifier struct {
	Recipient driver.Verifier
	HashInfo  exchange.HashInfo
}

func (cv *ClaimVerifier) Verify(tokenRequestAndTxID, claimSignature []byte) error {
	sig := &ClaimSignature{}
	err := json.Unmarshal(claimSignature, sig)
	if err != nil {
		return err
	}

	msg := concatTokenRequestTxIDPreimage(tokenRequestAndTxID, sig.Preimage)
	if err := cv.Recipient.Verify(msg, sig.RecipientSignature); err != nil {
		return err
	}

	hash := cv.HashInfo.HashFunc.New()
	if _, err = hash.Write(sig.Preimage); err != nil {
		return err
	}
	image := hash.Sum(nil)
	image = []byte(cv.HashInfo.HashEncoding.New().EncodeToString(image))

	if !bytes.Equal(cv.HashInfo.Hash, image) {
		return fmt.Errorf("hash mismatch: SHA(%x) = %x != %x", sig.Preimage, image, cv.HashInfo.Hash)
	}

	return nil
}

// ExchangeVerifier is the verifier of an exchange script signature
type ExchangeVerifier struct {
	Recipient driver.Verifier
	Sender    driver.Verifier
	Deadline  time.Time
	HashInfo  exchange.HashInfo
}

func (v *ExchangeVerifier) Verify(msg []byte, sigma []byte) error {
	if time.Now().Before(v.Deadline) {
		cv := &ClaimVerifier{Recipient: v.Recipient, HashInfo: exchange.HashInfo{
			Hash:         v.HashInfo.Hash,
			HashFunc:     v.HashInfo.HashFunc,
			HashEncoding: v.HashInfo.HashEncoding,
		}}
		if err := cv.Verify(msg, sigma); err != nil {
			return errors.WithMessagef(err, "failed verifying exchange claim signature")
		}
		return nil
	}
	if err := v.Sender.Verify(msg, sigma); err != nil {
		return errors.WithMessagef(err, "failed verifying exchange reclaim signature")
	}
	return nil
}
