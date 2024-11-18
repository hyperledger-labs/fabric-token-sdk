/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// TransferSignatureValidate validates the signatures for the inputs spent by an action
func TransferSignatureValidate(ctx *Context) error {
	ctx.InputTokens = ctx.TransferAction.InputTokens
	for _, tok := range ctx.InputTokens {
		ctx.Logger.Debugf("check sender [%s]", driver.Identity(tok.Owner.Raw).UniqueID())
		verifier, err := ctx.Deserializer.GetOwnerVerifier(tok.Owner.Raw)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%v][%s]", tok, driver.Identity(tok.Owner.Raw).UniqueID())
		}
		ctx.Logger.Debugf("signature verification [%v][%s]", tok, driver.Identity(tok.Owner.Raw).UniqueID())
		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(tok.Owner.Raw, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%v][%s]", tok, driver.Identity(tok.Owner.Raw).UniqueID())
		}
		ctx.Signatures = append(ctx.Signatures, sigma)
	}
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
		out := output.(*Output)
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
