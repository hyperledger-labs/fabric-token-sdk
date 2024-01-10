/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/pkg/errors"
)

func TransferPledgeValidate(ctx *Context) error {
	logger.Debug("pledge validation starts")
	for _, in := range ctx.InputTokens {
		identity, err := owner.UnmarshallTypedIdentity(in.Owner.Raw)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if identity.Type == pledge.ScriptType {
			if len(ctx.InputTokens) != 1 || len(ctx.Action.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: a pledge script only transfers the ownership of a token")
			}
			out := ctx.Action.GetOutputs()[0].(*Output)
			tok := out.Output
			sender, err := owner.UnmarshallTypedIdentity(ctx.InputTokens[0].Owner.Raw)
			if err != nil {
				return err
			}
			script := &pledge.Script{}
			err = json.Unmarshal(sender.Identity, script)
			if err != nil {
				return errors.Wrap(err, "failed to unmarshal pledge script")
			}
			if time.Now().Before(script.Deadline) {
				return errors.New("cannot reclaim pledge yet: wait for timeout to elapse.")
			}

			key, err := constructMetadataKey(ctx.Action)
			if err != nil {
				return errors.Wrap(err, "failed constructing metadata key")
			}

			if out.IsRedeem() {
				redeemKey := pledge.RedeemPledgeKey + key
				v, ok := ctx.Action.GetMetadata()[redeemKey]
				if !ok {
					return errors.Errorf("empty metadata of redeem for pledge script with identifier %s", redeemKey)
				}
				if v == nil {
					return errors.Errorf("invalid metadatata of redeem for pledge script with identifier %s, metadata should contain a proof", redeemKey)
				}
				ctx.CountMetadataKey(redeemKey)
				continue
			}
			if !script.Sender.Equal(tok.Owner.Raw) {
				return errors.Errorf("recipient of token does not correspond to the sender of reclaim request")
			}

			reclaimKey := pledge.MetadataReclaimKey + key
			v, ok := ctx.Action.GetMetadata()[reclaimKey]
			if !ok {
				return errors.Errorf("empty metadata of reclaim with identifier %s", reclaimKey)
			}
			if v == nil {
				return errors.Errorf("invalid metadatata of reclaim with identifier %s, metadata should contain a proof", reclaimKey)
			}
			ctx.CountMetadataKey(reclaimKey)
		}
	}

	for _, o := range ctx.Action.GetOutputs() {
		out, ok := o.(*Output)
		if !ok {
			return errors.Errorf("invalid output")
		}
		if out.IsRedeem() {
			continue
		}
		owner, err := owner.UnmarshallTypedIdentity(out.Output.Owner.Raw)
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
			v, ok := ctx.Action.GetMetadata()[pledge.MetadataKey+script.ID]
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

func constructMetadataKey(action *TransferAction) (string, error) {
	inputs, err := action.GetInputs()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve input IDs from action")
	}
	if len(inputs) != 1 {
		return "", errors.New("invalid transfer action, does not carry a single input")
	}
	prefix, components, err := keys.SplitCompositeKey(inputs[0])
	if err != nil {
		return "", errors.Wrapf(err, "unable to split input as key")
	}
	if prefix != keys.TokenKeyPrefix {
		return "", errors.Errorf("expected prefix [%s], got [%s], skipping", keys.TokenKeyPrefix, prefix)
	}
	txID := components[0]
	index, err := strconv.ParseUint(components[1], 10, 64)
	if err != nil {
		return "", errors.Errorf("invalid index for key [%s]", inputs[0])
	}
	return fmt.Sprintf(".%d.%s", index, txID), nil
}
