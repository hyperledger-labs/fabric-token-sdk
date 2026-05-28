/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
)

const (
	UnlockPreimage = "hashescrow.upi:"
	LockHashes     = "hashescrow.lh:"
	ClaimMetadata  = "hashescrow.cm:"
)

type LockMetadata struct {
	RecipientHash []byte `json:"recipient_hash"`
	SenderHash    []byte `json:"sender_hash"`
}

type ClaimMetadataValue struct {
	Preimage  []byte `json:"preimage"`
	ClaimedBy string `json:"claimed_by"`
}

func aggregateHash(recipientHash, senderHash []byte) []byte {
	h := sha256.New()
	h.Write(recipientHash)
	h.Write([]byte{':'})
	h.Write(senderHash)

	return h.Sum(nil)
}

func LockKey(recipientHash, senderHash []byte) string {
	return LockHashes + hex.EncodeToString(aggregateHash(recipientHash, senderHash))
}

func ClaimKey(recipientHash, senderHash []byte) string {
	return ClaimMetadata + hex.EncodeToString(aggregateHash(recipientHash, senderHash))
}

func LockValue(recipientHash, senderHash []byte) ([]byte, error) {
	raw, err := json.Marshal(&LockMetadata{
		RecipientHash: recipientHash,
		SenderHash:    senderHash,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal hash escrow lock metadata")
	}

	return raw, nil
}

func ClaimValue(preimage []byte, claimedBy string) ([]byte, error) {
	raw, err := json.Marshal(&ClaimMetadataValue{
		Preimage:  preimage,
		ClaimedBy: claimedBy,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal hash escrow claim metadata")
	}

	return raw, nil
}
