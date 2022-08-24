/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Context struct {
	PP                *PublicParams
	Deserializer      driver.Deserializer
	SignatureProvider driver.SignatureProvider
	Signatures        [][]byte
	InputTokens       []*token.Token
	Action            *TransferAction
	Ledger            driver.Ledger
}

// ValidateTransferFunc is the prototype of a validation function for a transfer action
type ValidateTransferFunc func(ctx *Context) error

// TransferSignatureValidate validates the signatures for the inputs spent by an action
func TransferSignatureValidate(ctx *Context) error {
	for _, tok := range ctx.InputTokens {
		logger.Debugf("check sender [%s]", view.Identity(tok.Owner.Raw).UniqueID())
		verifier, err := ctx.Deserializer.GetOwnerVerifier(tok.Owner.Raw)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%v][%s]", tok, view.Identity(tok.Owner.Raw).UniqueID())
		}
		logger.Debugf("signature verification [%v][%s]", tok, view.Identity(tok.Owner.Raw).UniqueID())
		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(tok.Owner.Raw, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%v][%s]", tok, view.Identity(tok.Owner.Raw).UniqueID())
		}
		ctx.Signatures = append(ctx.Signatures, sigma)
	}
	return nil
}

// TransferBalanceValidate checks that the sum of the inputs is equal to the sum of the outputs
func TransferBalanceValidate(ctx *Context) error {
	if ctx.Action.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	if len(ctx.InputTokens) == 0 {
		return errors.Errorf("there is no input")
	}
	if ctx.InputTokens[0] == nil {
		return errors.Errorf("first input is nil")
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
	for _, output := range ctx.Action.GetOutputs() {
		out := output.(*Output).Output
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

// TransferHTLCValidate checks the validity of the TransferHTLCValidate scripts, if any
func TransferHTLCValidate(ctx *Context) error {
	now := time.Now()

	for i, in := range ctx.InputTokens {
		owner, err := identity.UnmarshallRawOwner(in.Owner.Raw)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		// is it owned by a htlc script?
		if owner.Type == htlc.ScriptType {
			// Then, the first output must be compatible with this input.
			if len(ctx.Action.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: an htlc script only transfers the ownership of a token")
			}

			// check type and quantity
			out := ctx.Action.GetOutputs()[0].(*Output).Output
			if ctx.InputTokens[0].Type != out.Type {
				return errors.Errorf("invalid transfer action: type of input does not match type of output")
			}
			if ctx.InputTokens[0].Quantity != out.Quantity {
				return errors.Errorf("invalid transfer action: quantity of input does not match quantity of output")
			}

			// check owner field
			script, err := htlc2.VerifyOwner(ctx.InputTokens[0].Owner.Raw, out.Owner.Raw)
			if err != nil {
				return errors.Wrap(err, "failed to verify transfer from htlc script")
			}

			// check metadata
			sigma := ctx.Signatures[i]
			if err := HTLCMetadataCheck(ctx, script, sigma); err != nil {
				return errors.WithMessagef(err, "failed to check htlc metadata")
			}
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

		// if it is a htlc script than the deadline must be still valid
		owner, err := identity.UnmarshallRawOwner(out.Output.Owner.Raw)
		if err != nil {
			return err
		}
		if owner.Type == htlc.ScriptType {
			script := &htlc.Script{}
			err = json.Unmarshal(owner.Identity, script)
			if err != nil {
				return err
			}
			if script.Deadline.Before(now) {
				return errors.Errorf("htlc script invalid: expiration date has already passed")
			}
			continue
		}
	}
	return nil
}

// HTLCMetadataCheck checks that the HTLC metadata is in place
func HTLCMetadataCheck(ctx *Context, script *htlc.Script, sig []byte) error {
	claim := &htlc.ClaimSignature{}
	if err := json.Unmarshal(sig, claim); err != nil {
		return errors.Wrapf(err, "failed unmarshalling cliam signature")
	}
	if len(claim.Preimage) == 0 || len(claim.RecipientSignature) == 0 {
		return errors.New("expected a valid claim preImage and recipient signature")
	}

	if ctx.Action.NumOutputs() != 1 {
		return errors.Errorf("Number of outputs is %d", ctx.Action.NumOutputs())
	}
	out, ok := ctx.Action.GetOutputs()[0].(*Output)
	if !ok {
		return errors.Errorf("invalid transfer action output")
	}
	if out.IsRedeem() {
		return errors.Errorf("this is a redeem")
	}
	if !script.Recipient.Equal(out.Output.Owner.Raw) {
		return errors.Errorf("script recipient does not match the output owner")
	}

	if len(ctx.Action.Metadata) == 0 {
		return errors.Errorf("cannot find htlc pre-image, no metadata")
	}
	value, ok := ctx.Action.Metadata[htlc.ClaimPreImage]
	if !ok {
		return errors.Errorf("cannot find htlc pre-image, missing metadata entry")
	}
	if !bytes.Equal(value, claim.Preimage) {
		return errors.Errorf("invalid action, cannot match htlc pre-image with metadata [%x]!=[%x]", value, claim.Preimage)
	}

	return nil
}
