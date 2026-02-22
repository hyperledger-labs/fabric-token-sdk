/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"
	"context"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TransferActionValidate validates the transfer action
func TransferActionValidate(c context.Context, ctx *Context) error {
	return ctx.TransferAction.Validate()
}

// TransferSignatureValidate validates the signatures of the transfer action
func TransferSignatureValidate(c context.Context, ctx *Context) error {
	// recall that TransferActionValidate has been called before this function
	var signatures [][]byte

	if len(ctx.TransferAction.Inputs) == 0 {
		return ErrInvalidInputs
	}

	var isRedeem bool
	var inputToken []*token.Token
	for i, in := range ctx.TransferAction.Inputs {
		tok := in.Token
		inputToken = append(inputToken, tok)

		// check sender signature
		ctx.Logger.Debugf("check sender [%d][%s]", i, driver.Identity(tok.Owner).UniqueID())
		verifier, err := ctx.Deserializer.GetOwnerVerifier(c, tok.Owner)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%d][%v][%s]", i, in, driver.Identity(tok.Owner))
		}
		ctx.Logger.Debugf("signature verification [%d][%v][%s]", i, in, driver.Identity(tok.Owner).UniqueID())
		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(c, tok.Owner, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%d][%v][%s]", i, in, driver.Identity(tok.Owner))
		}
		signatures = append(signatures, sigma)
	}

	ctx.InputTokens = inputToken
	ctx.Signatures = signatures

	if len(ctx.PP.Issuers()) > 0 {
		// In this case we must ensure that an issuer signed as well if the action redeems tokens as well
		for _, output := range ctx.TransferAction.Outputs {
			if output.Owner == nil {
				isRedeem = true

				break
			}
		}

		if isRedeem {
			ctx.Logger.Debugf("action is a redeem, verify the signature of the issuer")
			issuer := ctx.TransferAction.GetIssuer()
			if issuer == nil {
				return ErrMissingIssuer
			}
			issuerVerifier, err := ctx.Deserializer.GetIssuerVerifier(c, issuer)
			if err != nil {
				return errors.Wrapf(err, "failed deserializing issuer [%s]", issuer.UniqueID())
			}
			sigma, err := ctx.SignatureProvider.HasBeenSignedBy(c, issuer, issuerVerifier)
			if err != nil {
				return errors.Wrapf(err, "failed signature verification [%s]", issuer.UniqueID())
			}
			ctx.Signatures = append(ctx.Signatures, sigma)
		}
	}

	return nil
}

// TransferUpgradeWitnessValidate validates that inputs that want to be upgraded, have a valid witness.
// It recomputes the commitment from the witness and compares it with the input token's data.
func TransferUpgradeWitnessValidate(c context.Context, ctx *Context) error {
	// recall that TransferActionValidate has been called before this function

	for _, input := range ctx.TransferAction.Inputs {
		witness := input.UpgradeWitness
		if witness != nil {
			// check that the corresponding input is compatible with the witness
			if witness.FabToken == nil {
				return ErrFabTokenNotFound
			}
			// recompute commitment to ensure the witness matches the input token's data
			// deserialize quantity witness.FabToken.Quantity
			q, err := token2.ToQuantity(witness.FabToken.Quantity, ctx.PP.QuantityPrecision)
			if err != nil {
				return errors.Wrapf(err, "failed to unmarshal quantity")
			}
			tokens, _, err := token.GetTokensWithWitnessAndBF([]uint64{q.ToBigInt().Uint64()}, []*math.Zr{witness.BlindingFactor}, witness.FabToken.Type, ctx.PP.PedersenGenerators, math.Curves[ctx.PP.Curve])
			if err != nil {
				return errors.Wrapf(err, "failed to compute commitment")
			}
			if !input.Token.Data.Equals(tokens[0]) {
				return ErrCommitmentMismatch
			}
			// check owner
			if !bytes.Equal(input.Token.Owner, witness.FabToken.Owner) {
				return ErrOwnersMismatch
			}
		}
	}

	return nil
}

// TransferZKProofValidate validates the ZK proof of the transfer action
func TransferZKProofValidate(c context.Context, ctx *Context) error {
	in := make([]*math.G1, len(ctx.InputTokens))
	for i, tok := range ctx.InputTokens {
		in[i] = tok.Data
	}

	if err := transfer.NewVerifier(
		in,
		ctx.TransferAction.GetOutputCommitments(),
		ctx.PP).Verify(ctx.TransferAction.GetProof()); err != nil {
		return errors.Join(err, ErrInvalidZKP)
	}

	return nil
}

// TransferHTLCValidate validates the HTLC scripts in the transfer action.
// It ensures that HTLC scripts only transfer ownership of a single token and that the script conditions are met.
func TransferHTLCValidate(c context.Context, ctx *Context) error {
	now := time.Now()

	for i, in := range ctx.InputTokens {
		owner, err := identity.UnmarshalTypedIdentity(in.Owner)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if owner.Type == htlc.ScriptType {
			// HTLC script must be a 1-to-1 transfer of ownership
			if len(ctx.InputTokens) != 1 || len(ctx.TransferAction.GetOutputs()) != 1 {
				return ErrInvalidHTLCAction
			}

			out, ok := ctx.TransferAction.GetOutputs()[0].(*token.Token)
			if !ok || out == nil {
				return ErrHTLCOutputNotFound
			}

			// check that owner field in output is correct based on HTLC script (lock, claim, or reclaim)
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

	for _, o := range ctx.TransferAction.Outputs {
		if o.IsRedeem() {
			continue
		}
		owner, err := identity.UnmarshalTypedIdentity(o.Owner)
		if err != nil {
			return err
		}
		if owner.Type == htlc.ScriptType {
			script := &htlc.Script{}
			err = script.FromBytes(owner.Identity)
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
