/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

// TransferHTLCValidate checks the validity of the HTLC scripts, if any
func TransferHTLCValidate[P driver.PublicParameters, T driver.Output, TA driver.TransferAction, IA driver.IssueAction](ctx *common.Context[P, T, TA, IA]) error {
	now := time.Now()

	for i, in := range ctx.InputTokens {
		owner, err := identity.UnmarshalTypedIdentity(in.GetOwner())
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		// is it owned by an htlc script?
		if owner.Type == htlc.ScriptType {
			// Then, the first output must be compatible with this input.
			if len(ctx.TransferAction.GetOutputs()) != 1 {
				return errors.New("invalid transfer action: an htlc script only transfers the ownership of a token")
			}

			// check it is not a redeem
			output := ctx.TransferAction.GetOutputs()[0]
			if output.IsRedeem() {
				return errors.New("invalid transfer action: the output corresponding to an htlc spending should not be a redeem")
			}

			// check that owner field in output is correct
			script, op, err := htlc.VerifyOwner(ctx.InputTokens[0].GetOwner(), output.GetOwner(), now)
			if err != nil {
				return errors.Wrap(err, "failed to verify transfer from htlc script")
			}

			// check metadata
			sigma := ctx.Signatures[i]
			metadataKey, err := htlc.MetadataClaimKeyCheck(ctx.TransferAction, script, op, sigma)
			if err != nil {
				return errors.WithMessagef(err, "failed to check htlc metadata")
			}
			if op != htlc.Reclaim {
				ctx.CountMetadataKey(metadataKey)
			}
		}
	}

	for _, out := range ctx.TransferAction.GetOutputs() {
		if out.IsRedeem() {
			continue
		}

		// if it is an htlc script then the deadline must still be valid
		owner, err := identity.UnmarshalTypedIdentity(out.GetOwner())
		if err != nil {
			return err
		}
		if owner.Type == htlc.ScriptType {
			script := &htlc.Script{}
			err = json.Unmarshal(owner.Identity, script)
			if err != nil {
				return err
			}
			if err := script.Validate(now); err != nil {
				return errors.WithMessagef(err, "htlc script invalid")
			}
			metadataKey, err := htlc.MetadataLockKeyCheck(ctx.TransferAction, script)
			if err != nil {
				return errors.WithMessagef(err, "failed to check htlc metadata")
			}
			ctx.CountMetadataKey(metadataKey)
			continue
		}
	}
	return nil
}
