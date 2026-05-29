/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"encoding/asn1"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = logging.MustGetLogger()

type Claim struct {
	TMSID    token.TMSID
	Wallet   string
	PreImage []byte
}

type ClaimView struct {
	*Claim
}

func (r *ClaimView) Call(ctx view.Context) (res any, err error) {
	var tx *hashescrow.Transaction
	defer func() {
		if e := recover(); e != nil {
			txID := "none"
			if tx != nil {
				txID = tx.ID()
			}
			if err == nil {
				err = errors.Errorf("<<<[%s]>>>: %s", txID, e)
			} else {
				err = errors.Errorf("<<<[%s]>>>: %s", txID, err)
			}
		}
	}()

	claimWallet := hashescrow.GetWallet(ctx, r.Wallet, token.WithTMSID(r.TMSID))
	assert.NotNil(claimWallet, "wallet [%s] not found", r.Wallet)

	var matched *token2.UnspentTokens
	runner := utils.NewRetryRunner(logger, 10, 2*time.Second, false)
	err = runner.RunWithContext(ctx.Context(), func() error {
		var err error
		matched, err = hashescrow.Wallet(claimWallet).ListByPreImage(ctx.Context(), r.PreImage)
		if err != nil {
			return errors.Wrap(err, "failed looking up hash escrow script")
		}
		if matched.Count() != 1 {
			return errors.Errorf("expected only one hash escrow script to match [%s], got [%d]", view.Identity(r.PreImage), matched.Count())
		}

		return nil
	})
	assert.NoError(err, "failed looking up hash escrow script")

	idProvider, err := id.GetProvider(ctx)
	assert.NoError(err, "failed getting id provider")
	tx, err = hashescrow.NewAnonymousTransaction(
		ctx,
		ttx.WithAuditor(idProvider.Identity("auditor")),
		ttx.WithTMSID(r.TMSID),
	)
	assert.NoError(err, "failed to create a hash escrow transaction")
	matchedToken := matched.At(0)
	script, err := scriptFromToken(matchedToken)
	assert.NoError(err, "failed reading hash escrow script for [%s]", matchedToken.Id)
	claimOwner, _, _, err := script.ResolveOwnerAndHashForPreimage(r.PreImage)
	assert.NoError(err, "failed resolving hash escrow claim owner for [%s]", matchedToken.Id)

	assert.NoError(tx.Claim(claimWallet, matchedToken, r.PreImage), "failed adding a hash escrow claim for [%s]", matchedToken.Id)

	_, err = ctx.RunView(ttx.NewCollectEndorsementsView(tx.Transaction))
	assert.NoError(err, "failed to collect endorsements on hash escrow transaction")

	assert.NoError(distributeClaimToCounterparty(ctx, tx, script, claimOwner), "failed distributing hash escrow claim transaction")

	_, err = ctx.RunView(ttx.NewOrderingAndFinalityView(tx.Transaction))
	assert.NoError(err, "failed to commit hash escrow transaction")

	return tx.ID(), nil
}

func scriptFromToken(tok *token2.UnspentToken) (*hashescrow.Script, error) {
	owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token owner")
	}
	if owner.Type != hashescrow.HashEscrow {
		return nil, errors.Errorf("invalid owner type, expected hash escrow script")
	}

	script := &hashescrow.Script{}
	if err := json.Unmarshal(owner.Identity, script); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal hash escrow script")
	}

	return script, nil
}

func distributeClaimToCounterparty(ctx view.Context, tx *hashescrow.Transaction, script *hashescrow.Script, claimOwner view.Identity) error {
	counterparty := script.Sender
	if claimOwner.Equal(script.Sender) {
		counterparty = script.Recipient
	}
	if counterparty.IsNone() || counterparty.Equal(claimOwner) {
		return nil
	}
	if _, err := tx.TokenService().WalletManager().OwnerWallet(ctx.Context(), counterparty); err == nil {
		return nil
	}

	txRaw, err := transactionBytesWithoutEnvelope(tx)
	if err != nil {
		return errors.Wrap(err, "failed marshalling claim transaction")
	}

	_, err = ctx.RunView(&ClaimDistributionView{
		Counterparty: counterparty,
		TxRaw:        txRaw,
	})
	if err != nil {
		return err
	}

	return nil
}

func transactionBytesWithoutEnvelope(tx *hashescrow.Transaction) ([]byte, error) {
	// The observer only needs token records; the submitter keeps the envelope for ordering.
	payload := tx.Payload
	envelope := payload.Envelope
	payload.Envelope = nil
	defer func() {
		payload.Envelope = envelope
	}()

	txRaw, err := tx.Bytes()
	if err != nil {
		return nil, err
	}

	if err := assertNoEnvelope(txRaw); err != nil {
		return nil, err
	}

	return txRaw, nil
}

type transactionSer struct {
	Nonce        []byte
	Creator      []byte
	ID           string
	Network      string
	Channel      string
	Namespace    string
	Signer       []byte
	Transient    []byte
	TokenRequest []byte
	Envelope     []byte
}

func assertNoEnvelope(txRaw []byte) error {
	var ser transactionSer
	if _, err := asn1.Unmarshal(txRaw, &ser); err != nil {
		return errors.Wrap(err, "failed checking serialized claim transaction")
	}
	if len(ser.Envelope) != 0 {
		return errors.Errorf("expected claim observer transaction without envelope, got envelope length [%d]", len(ser.Envelope))
	}

	return nil
}

func stripEnvelope(txRaw []byte) ([]byte, error) {
	var ser transactionSer
	if _, err := asn1.Unmarshal(txRaw, &ser); err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling serialized claim transaction")
	}
	if len(ser.Envelope) == 0 {
		return txRaw, nil
	}

	ser.Envelope = nil
	stripped, err := asn1.Marshal(ser)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling claim transaction without envelope")
	}

	return stripped, nil
}

type ClaimDistributionView struct {
	Counterparty view.Identity
	TxRaw        []byte
}

func (v *ClaimDistributionView) Call(ctx view.Context) (any, error) {
	session, err := ctx.GetSession(v, v.Counterparty)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting counterparty session")
	}

	if err := session.SendWithContext(ctx.Context(), v.TxRaw); err != nil {
		return nil, errors.Wrap(err, "failed sending claim transaction")
	}

	jsonSession := session2.NewFromSession(ctx, session)
	ack, err := jsonSession.ReceiveRawWithTimeout(time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "failed receiving claim transaction acknowledgement")
	}

	longTerm, _, _, err := endpoint.GetService(ctx).Resolve(ctx.Context(), v.Counterparty)
	if err != nil {
		return nil, errors.Wrap(err, "failed resolving counterparty long-term identity")
	}
	sigService, err := sig.GetService(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting signature service")
	}
	verifier, err := sigService.GetVerifier(longTerm)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting counterparty verifier")
	}
	if err := verifier.Verify(v.TxRaw, ack); err != nil {
		return nil, errors.Wrap(err, "failed verifying claim transaction acknowledgement")
	}

	return nil, nil
}

type ClaimAcceptView struct{}

func (h *ClaimAcceptView) Call(context view.Context) (any, error) {
	tx, err := receiveClaimTransaction(context)
	assert.NoError(err, "failed to receive hash escrow claim transaction")

	_, err = context.RunView(ttx.NewAcceptView(tx))
	assert.NoError(err, "failed to accept hash escrow claim transaction")

	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "hash escrow claim transaction was not committed")

	return tx.ID(), nil
}

func receiveClaimTransaction(context view.Context) (*ttx.Transaction, error) {
	txRaw, err := session2.JSON(context).ReceiveRawWithTimeout(4 * time.Minute)
	if err != nil {
		return nil, err
	}

	txRaw, err = stripEnvelope(txRaw)
	if err != nil {
		return nil, err
	}

	tx, err := ttx.NewTransactionFromBytes(context, txRaw)
	if err != nil {
		return nil, err
	}
	if err := tx.IsValid(context.Context()); err != nil {
		return nil, errors.Wrapf(err, "invalid transaction %s", tx.ID())
	}

	return tx, nil
}

type ClaimViewFactory struct{}

func (p *ClaimViewFactory) NewView(in []byte) (view.View, error) {
	f := &ClaimView{Claim: &Claim{}}
	err := json.Unmarshal(in, f.Claim)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
