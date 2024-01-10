/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Metadata struct {
	// OriginTokenID is the identifier of the pledged token in the origin network
	OriginTokenID *token.ID
	// OriginNetwork is the network where the pledge took place
	OriginNetwork string
}

type ProofOfTokenMetadataNonExistence struct {
	Origin   string
	TokenID  *token.ID
	Deadline time.Time
}

type ProofOfTokenMetadataExistence struct {
	Origin  string
	TokenID *token.ID
}

// ProveTokenExists queries whether a token with the given token ID exists
func (w *Translator) ProveTokenExists(tokenId *token.ID) error {
	key, err := keys.CreateTokenKey(tokenId.TxId, tokenId.Index)
	if err != nil {
		return err
	}
	tok, err := w.RWSet.GetState(w.namespace, key)
	if err != nil {
		return err
	}
	if tok == nil {
		return errors.Errorf("value at key [%s] is empty", tokenId)
	}
	key, err = keys.CreateProofOfExistenceKey(tokenId)
	if err != nil {
		return err
	}
	err = w.RWSet.SetState(w.namespace, key, tok)
	if err != nil {
		return err
	}
	return nil
}

// ProveTokenDoesNotExist queries whether a token with metadata including the given token ID and origin network does not exist
func (w *Translator) ProveTokenDoesNotExist(tokenID *token.ID, origin string, deadline time.Time) error {
	if time.Now().Before(deadline) {
		return errors.Errorf("deadline has not elapsed yet")
	}
	metadata, err := json.Marshal(&Metadata{OriginTokenID: tokenID, OriginNetwork: origin})
	if err != nil {
		return errors.Errorf("failed to marshal token metadata")
	}
	key, err := keys.CreateIssueActionMetadataKey(hash.Hashable(metadata).String())
	if err != nil {
		return err
	}
	tok, err := w.RWSet.GetState(w.namespace, key)
	if err != nil {
		return err
	}
	if tok != nil {
		return errors.Errorf("value at key [%s] is not empty", key)
	}
	proof := &ProofOfTokenMetadataNonExistence{Origin: origin, TokenID: tokenID, Deadline: deadline}
	raw, err := json.Marshal(proof)
	if err != nil {
		return err
	}
	key, err = keys.CreateProofOfNonExistenceKey(tokenID, origin)
	if err != nil {
		return err
	}
	err = w.RWSet.SetState(w.namespace, key, raw)
	if err != nil {
		return err
	}
	return nil
}

// ProveTokenWithMetadataExists queries whether a token with metadata including the given token ID and origin network exists
func (w *Translator) ProveTokenWithMetadataExists(tokenID *token.ID, origin string) error {
	metadata, err := json.Marshal(&Metadata{OriginTokenID: tokenID, OriginNetwork: origin})
	if err != nil {
		return errors.Errorf("failed to marshal token metadata")
	}
	key, err := keys.CreateIssueActionMetadataKey(hash.Hashable(metadata).String())
	if err != nil {
		return err
	}
	tok, err := w.RWSet.GetState(w.namespace, key)
	if err != nil {
		return err
	}
	if tok == nil {
		return errors.Errorf("value at key [%s] is empty", key)
	}
	proof := &ProofOfTokenMetadataExistence{Origin: origin, TokenID: tokenID}
	raw, err := json.Marshal(proof)
	if err != nil {
		return err
	}
	key, err = keys.CreateProofOfMetadataExistenceKey(tokenID, origin)
	if err != nil {
		return err
	}
	err = w.RWSet.SetState(w.namespace, key, raw)
	if err != nil {
		return err
	}
	return nil
}
