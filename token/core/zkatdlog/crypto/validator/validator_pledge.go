/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/pkg/errors"
)

func IssuePledgeValidate(ctx *Context) error {
	for k := range ctx.IssueAction.Metadata {
		ctx.CountMetadataKey(k)
	}
	return nil
}

func TransferPledgeValidate(ctx *Context) error {
	for _, in := range ctx.InputTokens {
		id, err := identity.UnmarshalTypedIdentity(in.Owner)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if id.Type == pledge.ScriptType {
			if len(ctx.InputTokens) != 1 || len(ctx.TransferAction.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: a pledge script only transfers the ownership of a token")
			}
			out := ctx.TransferAction.GetOutputs()[0].(*token.Token)
			sender, err := identity.UnmarshalTypedIdentity(ctx.InputTokens[0].Owner)
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
			if !script.Sender.Equal(out.Owner) {
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

	for _, o := range ctx.TransferAction.GetOutputs() {
		out, ok := o.(*token.Token)
		if !ok {
			return errors.Errorf("invalid output")
		}
		if out.IsRedeem() {
			continue
		}
		owner, err := identity.UnmarshalTypedIdentity(out.Owner)
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

func constructMetadataKey(action *transfer.TransferAction) (string, error) {
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
