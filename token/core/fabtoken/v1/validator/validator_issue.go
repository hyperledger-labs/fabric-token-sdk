/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func IssueValidate(c context.Context, ctx *Context) error {
	action := ctx.IssueAction

	if err := action.Validate(); err != nil {
		return errors.Wrapf(err, "failed validating issue action")
	}

	// verify that issue is valid
	if action.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	for _, output := range action.GetOutputs() {
		out := output.(*actions.Output)
		q, err := token.ToQuantity(out.Quantity, ctx.PP.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		zero := token.NewZeroQuantity(ctx.PP.QuantityPrecision)
		if q.Cmp(zero) == 0 {
			return errors.Errorf("quantity is zero")
		}
	}

	issuers := ctx.PP.IssuerIDs
	if len(issuers) != 0 {
		// check that issuer of this issue action is authorized
		found := false
		for _, issuer := range issuers {
			if bytes.Equal(action.Issuer, issuer) {
				found = true
				break
			}
		}
		if !found {
			return errors.Errorf("issuer [%s] is not in issuers", action.Issuer.String())
		}
	}

	// deserialize verifier for the issuer
	verifier, err := ctx.Deserializer.GetIssuerVerifier(action.Issuer)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for issuer identity [%s]", action.Issuer.String())
	}
	// verify if the token request concatenated with the anchor was signed by the issuer
	if _, err := ctx.SignatureProvider.HasBeenSignedBy(c, action.Issuer, verifier); err != nil {
		return errors.Wrapf(err, "failed verifying signature")
	}
	return nil
}
