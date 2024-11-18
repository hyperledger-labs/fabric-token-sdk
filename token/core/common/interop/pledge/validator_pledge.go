/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/pkg/errors"
)

func IssuePledgeValidate[P driver.PublicParameters, T driver.Output, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](ctx *common.Context[P, T, TA, IA, DS]) error {
	for k := range ctx.IssueAction.GetMetadata() {
		ctx.CountMetadataKey(k)
	}
	return nil
}

func TransferPledgeValidate[P driver.PublicParameters, T driver.Output, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](ctx *common.Context[P, T, TA, IA, DS]) error {
	for _, in := range ctx.InputTokens {
		id, err := identity.UnmarshalTypedIdentity(in.GetOwner())
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if id.Type == pledge.ScriptType {
			if len(ctx.InputTokens) != 1 || len(ctx.TransferAction.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: a pledge script only transfers the ownership of a token")
			}
			out := ctx.TransferAction.GetOutputs()[0]
			sender, err := identity.UnmarshalTypedIdentity(ctx.InputTokens[0].GetOwner())
			if err != nil {
				return err
			}
			script := &pledge.Script{}
			err = json.Unmarshal(sender.Identity, script)
			if err != nil {
				return err
			}
			if time.Now().Before(script.Deadline) {
				return errors.New("cannot reclaim pledge yet: wait for timeout to elapse.")
			}

			key, err := constructMetadataKey(ctx.TransferAction)
			if err != nil {
				return errors.Wrap(err, "failed constructing metadata key")
			}

			if out.IsRedeem() {
				redeemKey := pledge.RedeemPledgeKey + key
				v, ok := ctx.TransferAction.GetMetadata()[redeemKey]
				if !ok {
					return errors.Errorf("empty metadata of redeem for pledge script with identifier %s", redeemKey)
				}
				if v == nil {
					return errors.Errorf("invalid metadatata of redeem for pledge script with identifier %s, metadata should contain a proof", redeemKey)
				}
				ctx.CountMetadataKey(redeemKey)
				continue
			}
			if !script.Sender.Equal(out.GetOwner()) {
				return errors.New("recipient of token does not correspond to sender of reclaim request")
			}

			reclaimKey := pledge.MetadataReclaimKey + key
			v, ok := ctx.TransferAction.GetMetadata()[reclaimKey]
			if !ok {
				return errors.Errorf("empty metadata for pledge script with identifier %s", reclaimKey)
			}
			if v == nil {
				return errors.Errorf("invalid metadatata for pledge script with identifier %s, metadata should contain a proof", reclaimKey)
			}
			ctx.CountMetadataKey(reclaimKey)
		}
	}

	for _, out := range ctx.TransferAction.GetOutputs() {
		if out.IsRedeem() {
			continue
		}
		owner, err := identity.UnmarshalTypedIdentity(out.GetOwner())
		if err != nil {
			return err
		}
		if owner.Type == pledge.ScriptType {
			script := &pledge.Script{}
			err = json.Unmarshal(owner.Identity, script)
			if err != nil {
				return err
			}
			if script.Deadline.Before(time.Now()) {
				return errors.Errorf("pledge script is invalid: expiration date has already passed")
			}
			v, ok := ctx.TransferAction.GetMetadata()[pledge.MetadataKey+script.ID]
			if !ok {
				return errors.Errorf("empty metadata for pledge script with identifier %s", script.ID)
			}
			if !bytes.Equal(v, []byte("1")) {
				return errors.Errorf("invalid metadatata for pledge script with identifier %s", script.ID)
			}
			ctx.CountMetadataKey(pledge.MetadataKey + script.ID)
		}
	}
	return nil
}

func constructMetadataKey(action driver.TransferAction) (string, error) {
	inputs, err := action.GetInputs()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve input IDs from action")
	}
	if len(inputs) != 1 {
		return "", errors.New("invalid transfer action, does not carry a single input")
	}
	return fmt.Sprintf(".%d.%s", inputs[0].Index, inputs[0].TxId), nil
}
