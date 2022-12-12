/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

const (
	MetadataKey           = "metadata.pledge"
	defaultDeadlineOffset = time.Hour
)

func (t *Transaction) Pledge(wallet *token.OwnerWallet, destNetwork string, deadline time.Duration, recipient view.Identity, issuer view.Identity, typ string, value uint64) (string, error) {
	if deadline == 0 {
		deadline = defaultDeadlineOffset
	}
	if destNetwork == "" {
		return "", errors.Errorf("must specify a destination network")
	}
	if issuer.IsNone() {
		return "", errors.Errorf("must specify an issuer")
	}
	if recipient.IsNone() {
		return "", errors.Errorf("must specify a recipient")
	}
	pledgeID, err := generatePledgeID()
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate pledge ID")
	}
	me, err := wallet.GetRecipientIdentity()
	if err != nil {
		return "", err
	}
	script, err := t.recipientAsScript(me, destNetwork, deadline, recipient, issuer, pledgeID)
	if err != nil {
		return "", err
	}
	_, err = t.TokenRequest.Transfer(wallet, typ, []uint64{value}, []view.Identity{script}, token.WithTransferMetadata(MetadataKey+pledgeID, []byte("1")))
	return pledgeID, err
}

func (t *Transaction) recipientAsScript(sender view.Identity, destNetwork string, deadline time.Duration, recipient view.Identity, issuer view.Identity, pledgeID string) (view.Identity, error) {
	script := Script{
		Deadline:           time.Now().Add(deadline),
		DestinationNetwork: destNetwork,
		Recipient:          recipient,
		Issuer:             issuer,
		Sender:             sender,
		ID:                 pledgeID,
	}
	rawScript, err := json.Marshal(script)
	if err != nil {
		return nil, err
	}

	ro := &owner.TypedIdentity{
		Type:     ScriptType,
		Identity: rawScript,
	}
	return owner.MarshallTypedIdentity(ro)
}

// generatePledgeID generates a pledgeID randomly
func generatePledgeID() (string, error) {
	nonce, err := getRandomNonce()
	if err != nil {
		return "", errors.New("failed generating random nonce for pledgeID")
	}
	return hex.EncodeToString(nonce), nil
}

// getRandomNonce generates a random nonce using the package math/rand
func getRandomNonce() ([]byte, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}
	return key, nil
}
