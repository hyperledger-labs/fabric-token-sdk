/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"bytes"
	"crypto"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
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

// ClaimVerifier is the verifier of an exchange claim
type ClaimVerifier struct {
	Recipient    driver.Verifier
	Hash         []byte
	HashFunc     crypto.Hash
	HashEncoding encoding.Encoding
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

	hash := cv.HashFunc.New()
	if _, err = hash.Write(sig.Preimage); err != nil {
		return err
	}
	image := hash.Sum(nil)
	image = []byte(cv.HashEncoding.New().EncodeToString(image))

	if !bytes.Equal(cv.Hash, image) {
		return fmt.Errorf("hash mismatch: SHA(%x) = %x != %x", sig.Preimage, image, cv.Hash)
	}

	return nil
}

func concatTokenRequestTxIDPreimage(tokenRequestAndTxID []byte, preImage []byte) []byte {
	var msg []byte
	msg = append(msg, tokenRequestAndTxID...)
	msg = append(msg, preImage...)
	return msg
}
