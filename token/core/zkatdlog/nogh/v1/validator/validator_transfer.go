/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// TransferActionValidate validates the transfer action
func TransferActionValidate(ctx *Context) error {
	return ctx.TransferAction.Validate()
}

func TransferSignatureValidate(ctx *Context) error {
	var signatures [][]byte

	if len(ctx.TransferAction.Inputs) != len(ctx.TransferAction.InputTokens) {
		return errors.Errorf("invalid number of token inputs")
	}

	for i, in := range ctx.TransferAction.Inputs {
		tok := ctx.TransferAction.InputTokens[i]

		// TODO check witness

		// check sender signature
		ctx.Logger.Debugf("check sender [%d][%s]", i, driver.Identity(tok.Owner).UniqueID())
		verifier, err := ctx.Deserializer.GetOwnerVerifier(tok.Owner)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%d][%s][%s]", i, in, driver.Identity(tok.Owner).UniqueID())
		}
		ctx.Logger.Debugf("signature verification [%d][%s][%s]", i, in, driver.Identity(tok.Owner).UniqueID())
		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(tok.Owner, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%d][%s][%s]", i, in, driver.Identity(tok.Owner).UniqueID())
		}
		signatures = append(signatures, sigma)
	}

	ctx.InputTokens = ctx.TransferAction.InputTokens
	ctx.Signatures = signatures

	return nil
}

func TransferUpgradeWitnessValidate(ctx *Context) error {
	for i, witness := range ctx.TransferAction.InputUpgradeWitness {
		if witness != nil {
			// check that the corresponding input is compatible with the witness
			if witness.FabToken == nil {
				return errors.Errorf("fabtoken token not found in witness")
			}
			// recompute commitment
			// deserialize quantity witness.FabToken.Quantity
			q, err := token2.ToQuantity(witness.FabToken.Quantity, ctx.PP.QuantityPrecision)
			if err != nil {
				return errors.Wrapf(err, "failed to unmarshal quantity")
			}
			tokens, _, err := token.GetTokensWithWitness([]uint64{q.ToBigInt().Uint64()}, witness.FabToken.Type, ctx.PP.PedersenGenerators, math.Curves[ctx.PP.Curve])
			if err != nil {
				return errors.Wrapf(err, "failed to compute commitment")
			}
			if !ctx.TransferAction.InputTokens[i].Data.Equals(tokens[0]) {
				return errors.Wrapf(err, "recomputed commitment does not match")
			}
			// check owner
			if !bytes.Equal(ctx.TransferAction.InputTokens[i].Owner, witness.FabToken.Owner) {
				return errors.Errorf("owners do not correspond")
			}
		}
	}
	return nil
}

func TransferZKProofValidate(ctx *Context) error {
	in := make([]*math.G1, len(ctx.InputTokens))
	for i, tok := range ctx.InputTokens {
		in[i] = tok.Data
	}

	if err := transfer.NewVerifier(
		in,
		ctx.TransferAction.GetOutputCommitments(),
		ctx.PP).Verify(ctx.TransferAction.GetProof()); err != nil {
		return err
	}

	return nil
}

func TransferHTLCValidate(ctx *Context) error {
	now := time.Now()

	for i, in := range ctx.InputTokens {
		owner, err := identity.UnmarshalTypedIdentity(in.Owner)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if owner.Type == htlc.ScriptType {
			if len(ctx.InputTokens) != 1 || len(ctx.TransferAction.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: an htlc script only transfers the ownership of a token")
			}

			out := ctx.TransferAction.GetOutputs()[0].(*token.Token)

			// check that owner field in output is correct
			script, op, err := htlc2.VerifyOwner(ctx.InputTokens[0].Owner, out.Owner, now)
			if err != nil {
				return errors.Wrap(err, "failed to verify transfer from htlc script")
			}

			// check metadata
			sigma := ctx.Signatures[i]
			metadataKey, err := htlc2.MetadataClaimKeyCheck(ctx.TransferAction, script, op, sigma)
			if err != nil {
				return errors.WithMessagef(err, "failed to check htlc metadata")
			}
			if op != htlc2.Reclaim {
				ctx.CountMetadataKey(metadataKey)
			}
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
		if owner.Type == htlc.ScriptType {
			script := &htlc.Script{}
			err = json.Unmarshal(owner.Identity, script)
			if err != nil {
				return err
			}
			if err := script.Validate(now); err != nil {
				return errors.WithMessagef(err, "htlc script invalid")
			}
			metadataKey, err := htlc2.MetadataLockKeyCheck(ctx.TransferAction, script)
			if err != nil {
				return errors.WithMessagef(err, "failed to check htlc metadata")
			}
			ctx.CountMetadataKey(metadataKey)
			continue
		}
	}
	return nil
}
