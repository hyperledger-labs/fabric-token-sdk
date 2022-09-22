/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

type OperationType int

const (
	None OperationType = iota
	Claim
	Reclaim
)

type Action interface {
	GetMetadata() map[string][]byte
}

// VerifyOwner validates the owners of the transfer in the htlc script
func VerifyOwner(senderRawOwner []byte, outRawOwner []byte, now time.Time) (*htlc.Script, OperationType, error) {
	sender, err := identity.UnmarshallRawOwner(senderRawOwner)
	if err != nil {
		return nil, None, err
	}
	script := &htlc.Script{}
	err = json.Unmarshal(sender.Identity, script)
	if err != nil {
		return nil, None, err
	}

	if now.Before(script.Deadline) {
		// this should be a claim
		if !script.Recipient.Equal(outRawOwner) {
			return nil, None, errors.New("owner of output token does not correspond to recipient in htlc request")
		}
		return script, Claim, nil
	} else {
		// this should be a reclaim
		if !script.Sender.Equal(outRawOwner) {
			return nil, None, errors.New("owner of output token does not correspond to sender in htlc request")
		}
		return script, Reclaim, nil
	}
}

// MetadataCheck checks that the HTLC metadata is in place
func MetadataCheck(action Action, script *htlc.Script, op OperationType, sig []byte) error {
	if op == Reclaim {
		// No metadata in this case
		return nil
	}

	// Unmarshal signature to ClaimSignature
	claim := &htlc.ClaimSignature{}
	if err := json.Unmarshal(sig, claim); err != nil {
		return errors.Wrapf(err, "failed unmarshalling cliam signature [%s]", string(sig))
	}
	// Check that it is well-formed
	if len(claim.Preimage) == 0 || len(claim.RecipientSignature) == 0 {
		return errors.New("expected a valid claim preImage and recipient signature")
	}

	// Check that the pre-image is in the action's metadata
	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return errors.New("cannot find htlc pre-image, no metadata")
	}
	image, err := script.HashInfo.Image(claim.Preimage)
	if err != nil {
		return errors.Wrapf(err, "failed to compute image of [%x]", claim.Preimage)
	}
	value, ok := metadata[htlc.ClaimKey(image)]
	if !ok {
		return errors.New("cannot find htlc pre-image, missing metadata entry")
	}
	if !bytes.Equal(value, claim.Preimage) {
		return errors.Errorf("invalid action, cannot match htlc pre-image with metadata [%x]!=[%x]", value, claim.Preimage)
	}

	return nil
}
