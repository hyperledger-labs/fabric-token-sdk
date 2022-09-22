/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"
	"encoding/json"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

type Context struct {
	PP                *crypto.PublicParams
	Deserializer      driver.Deserializer
	SignatureProvider driver.SignatureProvider
	Signatures        [][]byte
	InputTokens       []*token.Token
	Action            *transfer.TransferAction
	Ledger            driver.Ledger
}

type ValidateTransferFunc func(ctx *Context) error

func TransferSignatureValidate(ctx *Context) error {
	var tokens []*token.Token
	var signatures [][]byte

	inputs, err := ctx.Action.GetInputs()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve inputs to spend")
	}
	for i, in := range inputs {
		logger.Debugf("load token [%d][%s]", i, in)
		bytes, err := ctx.Ledger.GetState(in)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve input to spend [%s]", in)
		}
		if len(bytes) == 0 {
			return errors.Errorf("input to spend [%s] does not exists", in)
		}

		tok := &token.Token{}
		if err := tok.Deserialize(bytes); err != nil {
			return errors.Wrapf(err, "failed to deserialize input to spend [%s]", in)
		}
		tokens = append(tokens, tok)
		logger.Debugf("check sender [%d][%s]", i, view.Identity(tok.Owner).UniqueID())
		verifier, err := ctx.Deserializer.GetOwnerVerifier(tok.Owner)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%d][%s][%s]", i, in, view.Identity(tok.Owner).UniqueID())
		}
		logger.Debugf("signature verification [%d][%s][%s]", i, in, view.Identity(tok.Owner).UniqueID())
		sigma, err := ctx.SignatureProvider.HasBeenSignedBy(tok.Owner, verifier)
		if err != nil {
			return errors.Wrapf(err, "failed signature verification [%d][%s][%s]", i, in, view.Identity(tok.Owner).UniqueID())
		}
		signatures = append(signatures, sigma)
	}

	ctx.InputTokens = tokens
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
		ctx.Action.GetOutputCommitments(),
		ctx.PP).Verify(ctx.Action.GetProof()); err != nil {
		return err
	}

	return nil
}

func TransferHTLCValidate(ctx *Context) error {
	now := time.Now()

	for i, in := range ctx.InputTokens {
		owner, err := identity.UnmarshallRawOwner(in.Owner)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if owner.Type == htlc.ScriptType {
			if len(ctx.InputTokens) != 1 || len(ctx.Action.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: an htlc script only transfers the ownership of a token")
			}

			out := ctx.Action.GetOutputs()[0].(*token.Token)

			// check that owner field in output is correct
			script, op, err := htlc2.VerifyOwner(ctx.InputTokens[0].Owner, out.Owner, now)
			if err != nil {
				return errors.Wrap(err, "failed to verify transfer from htlc script")
			}

			// check metadata
			sigma := ctx.Signatures[i]
			if err := HTLCMetadataCheck(ctx, script, op, sigma); err != nil {
				return errors.WithMessagef(err, "failed to check htlc metadata")
			}
		}
	}

	for _, o := range ctx.Action.GetOutputs() {
		out, ok := o.(*token.Token)
		if !ok {
			return errors.Errorf("invalid output")
		}
		if out.IsRedeem() {
			continue
		}
		owner, err := identity.UnmarshallRawOwner(out.Owner)
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
			continue
		}
	}
	return nil
}

// HTLCMetadataCheck checks that the HTLC metadata is in place
func HTLCMetadataCheck(ctx *Context, script *htlc.Script, op htlc2.OperationType, sig []byte) error {
	if op == htlc2.Reclaim {
		// No metadata in this case
		return nil
	}

	// Unmarshal signature to ClaimSignature
	claim := &htlc.ClaimSignature{}
	if err := json.Unmarshal(sig, claim); err != nil {
		return errors.Wrapf(err, "failed unmarshalling cliam signature [%s]", string(sig))
	}
	// Check that it is well-formed
	if len(claim.Preimage) == 0 || len(claim.RecipientSignature) == 0 {
		return errors.New("expected a valid claim preImage and recipient signature")
	}

	// Check the pre-image is in the action's metadata
	if len(ctx.Action.Metadata) == 0 {
		return errors.New("cannot find htlc pre-image, no metadata")
	}
	image, err := script.HashInfo.Image(claim.Preimage)
	if err != nil {
		return errors.Wrapf(err, "failed to compute image of [%x]", claim.Preimage)
	}
	value, ok := ctx.Action.Metadata[htlc.ClaimKey(image)]
	if !ok {
		return errors.New("cannot find htlc pre-image, missing metadata entry")
	}
	if !bytes.Equal(value, claim.Preimage) {
		return errors.Errorf("invalid action, cannot match htlc pre-image with metadata [%x]!=[%x]", value, claim.Preimage)
	}

	return nil
}
