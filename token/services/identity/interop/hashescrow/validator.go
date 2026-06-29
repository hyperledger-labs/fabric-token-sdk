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

const HashEscrow = ScriptType

// VerifyOwner validates that the output owner is either sender or recipient as encoded in the hash escrow script.
func VerifyOwner(senderRawOwner []byte, outRawOwner []byte) (*hashescrow.Script, error) {
	if len(outRawOwner) == 0 {
		return nil, errors.Errorf("the output owner must be set")
	}
	sender, err := identity.UnmarshalTypedIdentity(senderRawOwner)
	if err != nil {
		return nil, err
	}
	if sender.Type != HashEscrow {
		return nil, errors.Errorf("invalid identity type, expected [%s], got [%s]", HashEscrow, sender.Type)
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

func ClaimFromSignature(sig []byte) (*hashescrow.ClaimSignature, error) {
	claim := &hashescrow.ClaimSignature{}
	if err := json.Unmarshal(sig, claim); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling claim signature [%s]", string(sig))
	}
	if len(claim.Preimage) == 0 {
		return nil, errors.New("expected a valid claim preimage")
	}

	return claim, nil
}

func ResolveOwnerAndHash(script *hashescrow.Script, preimage []byte) ([]byte, []byte, string, error) {
	resolvedOwner, resolvedHash, claimedBy, err := script.ResolveOwnerAndHashForPreimage(preimage)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "failed resolving recipient from preimage")
	}

	return resolvedOwner, resolvedHash, claimedBy, nil
}

// ClaimMetadataCheck validates claim metadata and returns the matched metadata key.
func ClaimMetadataCheck(action Action, script *hashescrow.Script, preimage []byte, claimedBy string) (string, error) {
	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return "", errors.New("cannot find hash escrow unlock preimage, no metadata")
	}
	key := hashescrow.ClaimKey(script.RecipientHashInfo.Hash, script.SenderHashInfo.Hash)
	value, ok := metadata[key]
	if !ok {
		return "", errors.New("cannot find hash escrow unlock preimage, missing metadata entry")
	}
	claimValue := &hashescrow.ClaimMetadataValue{}
	if err := json.Unmarshal(value, claimValue); err != nil {
		return "", errors.Wrapf(err, "cannot unmarshal hash escrow claim metadata value [%s]", string(value))
	}
	if !bytes.Equal(claimValue.Preimage, preimage) {
		return "", errors.Errorf("invalid action, cannot match hash escrow unlock preimage with metadata [%x]!=[%x]", claimValue.Preimage, preimage)
	}
	if claimValue.ClaimedBy != claimedBy {
		return "", errors.Errorf("invalid action, cannot match hash escrow claimant side with metadata [%s]!=[%s]", claimValue.ClaimedBy, claimedBy)
	}

	return key, nil
}

// LockMetadataCheck validates lock metadata and returns the lock metadata key.
func LockMetadataCheck(action Action, script *hashescrow.Script) (string, error) {
	metadata := action.GetMetadata()
	if len(metadata) == 0 {
		return "", errors.New("cannot find hash escrow lock, no metadata")
	}
	key := hashescrow.LockKey(script.RecipientHashInfo.Hash, script.SenderHashInfo.Hash)
	lockValue, ok := metadata[key]
	if !ok {
		return "", errors.New("cannot find hash escrow lock, missing metadata entry")
	}
	lockMetadata := &hashescrow.LockMetadata{}
	if err := json.Unmarshal(lockValue, lockMetadata); err != nil {
		return "", errors.Wrapf(err, "cannot unmarshal hash escrow lock metadata value [%s]", string(lockValue))
	}
	if !bytes.Equal(lockMetadata.RecipientHash, script.RecipientHashInfo.Hash) {
		return "", errors.Errorf("invalid action, cannot match recipient hash escrow lock with metadata [%x]!=[%x]", lockMetadata.RecipientHash, script.RecipientHashInfo.Hash)
	}
	if !bytes.Equal(lockMetadata.SenderHash, script.SenderHashInfo.Hash) {
		return "", errors.Errorf("invalid action, cannot match sender hash escrow lock with metadata [%x]!=[%x]", lockMetadata.SenderHash, script.SenderHashInfo.Hash)
	}

	return key, nil
}
