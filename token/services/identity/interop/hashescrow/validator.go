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
func MetadataClaimKeyCheck(action Action, script *hashescrow.Script, sig []byte) (string, []byte, error) {
	claim := &hashescrow.ClaimSignature{}
	if err := json.Unmarshal(sig, claim); err != nil {
		return "", nil, errors.Wrapf(err, "failed unmarshalling claim signature [%s]", string(sig))
	}
	if len(claim.Preimage) == 0 {
		return "", nil, errors.New("expected a valid claim preImage")
	}
	resolvedOwner, image, err := script.ResolveRecipientForPreImage(claim.Preimage)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed resolving recipient from preimage")
	}

	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return "", nil, errors.New("cannot find hash escrow pre-image, no metadata")
	}
	key := hashescrow.ClaimKey(image)
	value, ok := metadata[key]
	if !ok {
		return "", nil, errors.New("cannot find hash escrow pre-image, missing metadata entry")
	}
	if !bytes.Equal(value, claim.Preimage) {
		return "", nil, errors.Errorf("invalid action, cannot match hash escrow pre-image with metadata [%x]!=[%x]", value, claim.Preimage)
	}

	return key, resolvedOwner, nil
}

// MetadataLockKeyCheck validates lock metadata and returns the lock metadata key.
func MetadataLockKeyCheck(action Action, script *hashescrow.Script) ([]string, error) {
	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return nil, errors.New("cannot find hash escrow lock, no metadata")
	}
	recipientKey := hashescrow.LockKey(script.RecipientHashInfo.Hash)
	recipientValue, ok := metadata[recipientKey]
	if !ok {
		return nil, errors.New("cannot find recipient hash escrow lock, missing metadata entry")
	}
	if !bytes.Equal(recipientValue, hashescrow.LockValue(script.RecipientHashInfo.Hash)) {
		return nil, errors.Errorf("invalid action, cannot match recipient hash escrow lock with metadata [%x]!=[%x]", recipientValue, script.RecipientHashInfo.Hash)
	}
	senderKey := hashescrow.LockKey(script.SenderHashInfo.Hash)
	senderValue, ok := metadata[senderKey]
	if !ok {
		return nil, errors.New("cannot find sender hash escrow lock, missing metadata entry")
	}
	if !bytes.Equal(senderValue, hashescrow.LockValue(script.SenderHashInfo.Hash)) {
		return nil, errors.Errorf("invalid action, cannot match sender hash escrow lock with metadata [%x]!=[%x]", senderValue, script.SenderHashInfo.Hash)
	}

	return []string{recipientKey, senderKey}, nil
}
