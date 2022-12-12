/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	RedeemPledgeKey = "metadata.redeemPledge"
)

// RedeemPledge appends a redeem action to the request. The action will be prepared using the provided owner wallet.
// The action redeems the passed token.
func (t *Transaction) RedeemPledge(wallet *token.OwnerWallet, tok *token2.UnspentToken, tokenID *token2.ID, proof []byte) error {
	if proof == nil {
		return errors.New("must provide proof")
	}
	if tokenID == nil {
		return errors.New("must provide token ID")
	}

	q, err := token2.ToQuantity(tok.Quantity, t.TokenRequest.TokenService.PublicParametersManager().PublicParameters().Precision())
	if err != nil {
		return errors.Wrapf(err, "failed to convert quantity [%s]", tok.Quantity)
	}

	owner, err := owner.UnmarshallTypedIdentity(tok.Owner.Raw)
	if err != nil {
		return err
	}
	script := &Script{}
	if owner.Type != ScriptType {
		return errors.Errorf("invalid owner type, expected a pledge script")
	}

	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return errors.Errorf("failed to unmarshal TypedIdentity as a pledge script")
	}

	// Register the signer for the redeem
	sigService := t.TokenService().SigService()
	signer, err := sigService.GetSigner(script.Issuer)
	if err != nil {
		return err
	}
	verifier, err := sigService.OwnerVerifier(script.Issuer)
	if err != nil {
		return err
	}
	// TODO: script.Issues is an owner identity, shall we switch to issuer identity?
	if err != nil {
		return err
	}
	redeemSigner := &Signer{Issuer: signer}
	redeemVerifier := &Verifier{
		Issuer: verifier,
	}
	logger.Debugf("registering signer for redeem...")
	if err := sigService.RegisterSigner(
		tok.Owner.Raw,
		redeemSigner,
		redeemVerifier,
	); err != nil {
		return err
	}

	if err := view2.GetEndpointService(t.SP).Bind(script.Issuer, tok.Owner.Raw); err != nil {
		return err
	}

	proofKey := RedeemPledgeKey + fmt.Sprintf(".%d.%s", tokenID.Index, tokenID.TxId)

	err = t.TokenRequest.Redeem(wallet, tok.Type, q.ToBigInt().Uint64(), token.WithTokenIDs(tok.Id), token.WithTransferMetadata(proofKey, proof))
	return err
}
