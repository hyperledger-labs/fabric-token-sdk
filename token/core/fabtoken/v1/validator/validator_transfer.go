/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// TransferActionValidate validates the transfer action
func TransferActionValidate(ctx *Context) error {
	return ctx.TransferAction.Validate()
}

// TransferSignatureValidate validates the signatures for the inputs spent by an action
func TransferSignatureValidate(ctx *Context) error {
	if len(ctx.TransferAction.Inputs) == 0 {
		return errors.Errorf("invalid number of token inputs, expected at least 1")
	}

	var inputToken []*actions.Output
	for _, in := range ctx.TransferAction.Inputs {
		tok := in.Input

		inputToken = append(inputToken, tok)
		owner := tok.GetOwner()
		ctx.Logger.Debugf("check sender [%s]", driver.Identity(owner).UniqueID())
		verifier, err := ctx.Deserializer.GetOwnerVerifier(owner)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%v][%s]", tok, driver.Identity(owner).UniqueID())
		}
		ctx.Logger.Debugf("signature verification [%v][%s]", tok, driver.Identity(owner).UniqueID())

		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(owner, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%v][%s]", tok, driver.Identity(owner).UniqueID())
		}
		ctx.Signatures = append(ctx.Signatures, sigma)
	}

	var isRedeem bool
	for _, output := range ctx.TransferAction.Outputs {
		if output.Owner == nil {
			isRedeem = true
			break
		}
	}

	// If transfer action is a redeem, verify the signature of the issuer
	if isRedeem {
		ctx.Logger.Infof("action is a redeem, verify the signature of the issuer")

		issuer := ctx.TransferAction.GetIssuer()
		if issuer == nil {
			return errors.Errorf("On Redeem action, must have at least one issuer")
		}

		issuerVerifier, err := ctx.Deserializer.GetIssuerVerifier(issuer)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing issuer [%s]", issuer.UniqueID())
		}

		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(issuer, issuerVerifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%s]", issuer.UniqueID())
		}
		ctx.Signatures = append(ctx.Signatures, sigma)
	}

	ctx.InputTokens = inputToken
	return nil
}

// TransferBalanceValidate checks that the sum of the inputs is equal to the sum of the outputs
func TransferBalanceValidate(ctx *Context) error {
	if ctx.TransferAction.NumOutputs() == 0 {
		return errors.New("there is no output")
	}
	if len(ctx.InputTokens) == 0 {
		return errors.New("there is no input")
	}
	if ctx.InputTokens[0] == nil {
		return errors.New("first input is nil")
	}
	typ := ctx.InputTokens[0].Type
	inputSum := token.NewZeroQuantity(ctx.PP.QuantityPrecision)
	outputSum := token.NewZeroQuantity(ctx.PP.QuantityPrecision)
	for i, input := range ctx.InputTokens {
		if input == nil {
			return errors.Errorf("input %d is nil", i)
		}
		q, err := token.ToQuantity(input.Quantity, ctx.PP.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", input.Quantity)
		}
		inputSum.Add(q)
		// check that all inputs have the same type
		if input.Type != typ {
			return errors.Errorf("input type %s does not match type %s", input.Type, typ)
		}
	}
	for _, output := range ctx.TransferAction.GetOutputs() {
		out := output.(*actions.Output)
		q, err := token.ToQuantity(out.Quantity, ctx.PP.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		outputSum.Add(q)
		// check that all outputs have the same type, and it is the same type as inputs
		if out.Type != typ {
			return errors.Errorf("output type %s does not match type %s", out.Type, typ)
		}
	}
	// check equality of sum of inputs and outputs
	if inputSum.Cmp(outputSum) != 0 {
		return errors.Errorf("input sum %v does not match output sum %v", inputSum, outputSum)
	}

	return nil
}

// TransferHTLCValidate checks the validity of the HTLC scripts, if any
func TransferHTLCValidate(ctx *Context) error {
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

			// check type and quantity
			output := ctx.TransferAction.GetOutputs()[0].(*actions.Output)
			tok := output
			if ctx.InputTokens[0].Type != tok.Type {
				return errors.New("invalid transfer action: type of input does not match type of output")
			}
			if ctx.InputTokens[0].Quantity != tok.Quantity {
				return errors.New("invalid transfer action: quantity of input does not match quantity of output")
			}
			if output.IsRedeem() {
				return errors.New("invalid transfer action: the output corresponding to an htlc spending should not be a redeem")
			}

			// check owner field
			script, op, err := htlc2.VerifyOwner(ctx.InputTokens[0].GetOwner(), tok.Owner, now)
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
		out, ok := o.(*actions.Output)
		if !ok {
			return errors.New("invalid output")
		}
		if out.IsRedeem() {
			continue
		}

		// if it is an htlc script then the deadline must still be valid
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

// TransferApplicationDataValidate accepts any metadata in the "pub" namespace.
// This gives the user of the Token SDK the option to attach public data to the token transaction.
func TransferApplicationDataValidate(ctx *Context) error {
	for key := range ctx.TransferAction.Metadata {
		if strings.HasPrefix(key, "pub.") {
			ctx.CountMetadataKey(key)
		}
	}
	return nil
}
