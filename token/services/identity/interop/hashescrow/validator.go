/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"bytes"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
)

//go:generate counterfeiter -o mock/action.go -fake-name Action . Action
type Action interface {
	GetMetadata() map[string][]byte
}

// VerifyOwner validates that the output owner is either sender or recipient as encoded in the hash escrow script.
func VerifyOwner(senderRawOwner []byte, outRawOwner []byte) (*hashescrow.Script, error) {
	if len(outRawOwner) == 0 {
		return nil, errors.Errorf("the output owner must be set")
	}
	sender, err := identity.UnmarshalTypedIdentity(senderRawOwner)
	if err != nil {
		return nil, err
	}
	if sender.Type != ScriptType {
		return nil, errors.Errorf("invalid identity type, expected [%s], got [%s]", ScriptType, sender.Type)
	}
	script := &hashescrow.Script{}
	err = json.Unmarshal(sender.Identity, script)
	if err != nil {
		return nil, err
	}
	if script.Recipient.Equal(outRawOwner) || script.Sender.Equal(outRawOwner) {
		return script, nil
	}

	return nil, errors.New("owner of output token does not correspond to sender or recipient in hash escrow request")
}

// MetadataClaimKeyCheck validates claim metadata and returns the matched metadata key.
func MetadataClaimKeyCheck(action Action, script *hashescrow.Script, sig []byte) (string, error) {
	claim := &hashescrow.ClaimSignature{}
	if err := json.Unmarshal(sig, claim); err != nil {
		return "", errors.Wrapf(err, "failed unmarshalling claim signature [%s]", string(sig))
	}
	if len(claim.Preimage) == 0 || len(claim.ClaimantSignature) == 0 {
		return "", errors.New("expected a valid claim preImage and claimant signature")
	}

	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return "", errors.New("cannot find hash escrow pre-image, no metadata")
	}
	image, err := script.HashInfo.Image(claim.Preimage)
	if err != nil {
		return "", errors.Wrapf(err, "failed to compute image of [%x]", claim.Preimage)
	}
	key := hashescrow.ClaimKey(image)
	value, ok := metadata[key]
	if !ok {
		return "", errors.New("cannot find hash escrow pre-image, missing metadata entry")
	}
	if !bytes.Equal(value, claim.Preimage) {
		return "", errors.Errorf("invalid action, cannot match hash escrow pre-image with metadata [%x]!=[%x]", value, claim.Preimage)
	}

	return key, nil
}

// MetadataLockKeyCheck validates lock metadata and returns the lock metadata key.
func MetadataLockKeyCheck(action Action, script *hashescrow.Script) (string, error) {
	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return "", errors.New("cannot find hash escrow lock, no metadata")
	}
	key := hashescrow.LockKey(script.HashInfo.Hash)
	value, ok := metadata[key]
	if !ok {
		return "", errors.New("cannot find hash escrow lock, missing metadata entry")
	}
	if !bytes.Equal(value, hashescrow.LockValue(script.HashInfo.Hash)) {
		return "", errors.Errorf("invalid action, cannot match hash escrow lock with metadata [%x]!=[%x]", value, script.HashInfo.Hash)
	}

	return key, nil
}
