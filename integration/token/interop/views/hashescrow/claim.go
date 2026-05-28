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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
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
	assert.NoError(tx.Claim(claimWallet, matched.At(0), r.PreImage), "failed adding a hash escrow claim for [%s]", matched.At(0).Id)

	_, err = ctx.RunView(ttx.NewCollectEndorsementsView(tx.Transaction))
	assert.NoError(err, "failed to collect endorsements on hash escrow transaction")

	_, err = ctx.RunView(ttx.NewOrderingAndFinalityView(tx.Transaction))
	assert.NoError(err, "failed to commit hash escrow transaction")

	return tx.ID(), nil
}

type ClaimViewFactory struct{}

func (p *ClaimViewFactory) NewView(in []byte) (view.View, error) {
	f := &ClaimView{Claim: &Claim{}}
	err := json.Unmarshal(in, f.Claim)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
