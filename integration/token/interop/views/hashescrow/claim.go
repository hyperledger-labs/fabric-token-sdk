/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
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

	assert.NoError(distributeClaimToCounterparty(ctx, r, tx, script, claimOwner), "failed distributing hash escrow claim transaction")

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

func distributeClaimToCounterparty(ctx view.Context, initiator view.View, tx *hashescrow.Transaction, script *hashescrow.Script, claimOwner view.Identity) error {
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

	session, err := ctx.GetSession(initiator, counterparty)
	if err != nil {
		return errors.Wrap(err, "failed getting counterparty session")
	}

	txRaw, err := tx.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed marshalling claim transaction")
	}

	if err := session.SendWithContext(ctx.Context(), txRaw); err != nil {
		return errors.Wrap(err, "failed sending claim transaction")
	}

	jsonSession := session2.NewFromSession(ctx, session)
	ack, err := jsonSession.ReceiveRawWithTimeout(time.Minute)
	if err != nil {
		return errors.Wrap(err, "failed receiving claim transaction acknowledgement")
	}

	longTerm, _, _, err := endpoint.GetService(ctx).Resolve(ctx.Context(), counterparty)
	if err != nil {
		return errors.Wrap(err, "failed resolving counterparty long-term identity")
	}
	sigService, err := sig.GetService(ctx)
	if err != nil {
		return errors.Wrap(err, "failed getting signature service")
	}
	verifier, err := sigService.GetVerifier(longTerm)
	if err != nil {
		return errors.Wrap(err, "failed getting counterparty verifier")
	}
	if err := verifier.Verify(txRaw, ack); err != nil {
		return errors.Wrap(err, "failed verifying claim transaction acknowledgement")
	}

	return nil
}

type ClaimAcceptView struct{}

func (h *ClaimAcceptView) Call(context view.Context) (any, error) {
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive hash escrow claim transaction")

	_, err = context.RunView(ttx.NewAcceptView(tx))
	assert.NoError(err, "failed to accept hash escrow claim transaction")

	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "hash escrow claim transaction was not committed")

	return tx.ID(), nil
}

type ClaimViewFactory struct{}

func (p *ClaimViewFactory) NewView(in []byte) (view.View, error) {
	f := &ClaimView{Claim: &Claim{}}
	err := json.Unmarshal(in, f.Claim)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
