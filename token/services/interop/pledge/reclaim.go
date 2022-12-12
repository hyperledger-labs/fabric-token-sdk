/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	MetadataReclaimKey = "metadata.reclaim"
)

func (t *Transaction) Reclaim(wallet *token.OwnerWallet, tok *token2.UnspentToken, issuerSignature []byte, tokenID *token2.ID, proof []byte) error {
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

	if owner.Type != ScriptType {
		return errors.Errorf("invalid owner type, expected a pledge script")
	}

	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return errors.Errorf("failed to unmarshal TypedIdentity as a pledge script")
	}

	// Register the signer for the reclaim
	sigService := t.TokenService().SigService()
	signer, err := sigService.GetSigner(script.Sender)
	if err != nil {
		return err
	}
	verifier, err := sigService.OwnerVerifier(script.Sender)
	if err != nil {
		return err
	}
	// TODO: script.Issues is an owner identity, shall we switch to issuer identity?
	issuer, err := sigService.OwnerVerifier(script.Issuer)
	if err != nil {
		return err
	}
	reclaimSigner := &Signer{Sender: signer, IssuerSignature: issuerSignature}
	reclaimVerifier := &Verifier{
		Sender:   verifier,
		Issuer:   issuer,
		PledgeID: script.ID,
	}
	logger.Debugf("registering signer for reclaim...")
	if err := sigService.RegisterSigner(
		tok.Owner.Raw,
		reclaimSigner,
		reclaimVerifier,
	); err != nil {
		return err
	}

	if err := view2.GetEndpointService(t.SP).Bind(script.Sender, tok.Owner.Raw); err != nil {
		return err
	}

	proofKey := MetadataReclaimKey + fmt.Sprintf(".%d.%s", tokenID.Index, tokenID.TxId)

	_, err = t.TokenRequest.Transfer(wallet, tok.Type, []uint64{q.ToBigInt().Uint64()}, []view.Identity{script.Sender}, token.WithTokenIDs(tok.Id), token.WithTransferMetadata(proofKey, proof))
	return err
}
