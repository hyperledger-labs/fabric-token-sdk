/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
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

// ValidateTransferFunc is the prototype of a validation function for a transfer action
type ValidateTransferFunc func(inputTokens []*token.Token, tr driver.TransferAction, signatureProvider driver.SignatureProvider) error

// TransferSignature validates the signatures for the inputs spent by an action
type TransferSignature struct {
	Deserializer driver.Deserializer
}

// Validate validates the signatures for the inputs spent by an action
func (v *TransferSignature) Validate(inputTokens []*token.Token, tr driver.TransferAction, signatureProvider driver.SignatureProvider) error {
	for _, tok := range inputTokens {
		logger.Debugf("check sender [%s]", view.Identity(tok.Owner.Raw).UniqueID())
		verifier, err := v.Deserializer.GetOwnerVerifier(tok.Owner.Raw)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%v][%s]", tok, view.Identity(tok.Owner.Raw).UniqueID())
		}
		logger.Debugf("signature verification [%v][%s]", tok, view.Identity(tok.Owner.Raw).UniqueID())
		_, err = signatureProvider.HasBeenSignedBy(tok.Owner.Raw, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%v][%s]", tok, view.Identity(tok.Owner.Raw).UniqueID())
		}
	}
	return nil
}

// TransferBalance checks that the sum of the inputs is equal to the sum of the ouputs
type TransferBalance struct {
	PP *PublicParams
}

// Validate checks that the sum of the inputs is equal to the sum of the ouputs
func (v *TransferBalance) Validate(inputTokens []*token.Token, tr driver.TransferAction, signatureProvider driver.SignatureProvider) error {
	if tr.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	if len(inputTokens) == 0 {
		return errors.Errorf("there is no input")
	}
	if inputTokens[0] == nil {
		return errors.Errorf("first input is nil")
	}
	typ := inputTokens[0].Type
	inputSum := token.NewZeroQuantity(v.PP.QuantityPrecision)
	outputSum := token.NewZeroQuantity(v.PP.QuantityPrecision)
	for i, input := range inputTokens {
		if input == nil {
			return errors.Errorf("input %d is nil", i)
		}
		q, err := token.ToQuantity(input.Quantity, v.PP.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", input.Quantity)
		}
		inputSum.Add(q)
		// check that all inputs have the same type
		if input.Type != typ {
			return errors.Errorf("input type %s does not match type %s", input.Type, typ)
		}
	}
	for _, output := range tr.GetOutputs() {
		out := output.(*Output).Output
		q, err := token.ToQuantity(out.Quantity, v.PP.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		outputSum.Add(q)
		// check that all outputs have the same type and it is the same type as inputs
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

// TransferHTLC checks the validity of the TransferHTLC scripts, if any
func TransferHTLC(inputTokens []*token.Token, tr driver.TransferAction, signatureProvider driver.SignatureProvider) error {
	now := time.Now()

	for _, in := range inputTokens {
		owner, err := identity.UnmarshallRawOwner(in.Owner.Raw)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		// is it owner by an htlc script?
		if owner.Type == htlc.ScriptType {
			// Then, the first output must be compatible with this input.
			if len(tr.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: an htlc script only transfers the ownership of a token")
			}

			// check type and quantity
			out := tr.GetOutputs()[0].(*Output).Output
			if inputTokens[0].Type != out.Type {
				return errors.Errorf("invalid transfer action: type of input does not match type of output")
			}
			if inputTokens[0].Quantity != out.Quantity {
				return errors.Errorf("invalid transfer action: quantity of input does not match quantity of output")
			}

			// check owner field
			if err := htlc2.VerifyOwner(inputTokens[0].Owner.Raw, out.Owner.Raw); err != nil {
				return errors.Wrap(err, "failed to verify transfer from htlc script")
			}
		}
	}

	for _, o := range tr.GetOutputs() {
		out, ok := o.(*Output)
		if !ok {
			return errors.Errorf("invalid output")
		}
		if out.IsRedeem() {
			continue
		}

		// if it is an htlc script than the deadline must be still valid
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
