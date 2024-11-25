/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

func TransferSignatureValidate(ctx *Context) error {
	var signatures [][]byte

	if len(ctx.TransferAction.Inputs) != len(ctx.TransferAction.InputTokens) {
		return errors.Errorf("invalid number of token inputs")
	}

	for i, in := range ctx.TransferAction.Inputs {
		tok := ctx.TransferAction.InputTokens[i]
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

func TransferZKProofValidate(ctx *Context) error {
	in := make([]*math.G1, len(ctx.InputTokens))
	for i, tok := range ctx.InputTokens {
		in[i] = tok.GetCommitment()
	}

	if err := transfer.NewVerifier(
		in,
		ctx.TransferAction.GetOutputCommitments(),
		ctx.PP).Verify(ctx.TransferAction.GetProof()); err != nil {
		return err
	}

	return nil
}
