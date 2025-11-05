/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var (
	ErrAuditorSignaturesMissing = errors.New("auditor signatures missing")
	ErrAuditorSignaturesPresent = errors.New("auditor signatures present")
)

func AuditingSignaturesValidate[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](c context.Context, ctx *Context[P, T, TA, IA, DS]) error {
	if len(ctx.PP.Auditors()) == 0 {
		// enforce no auditor signatures are attached
		if len(ctx.TokenRequest.AuditorSignatures) != 0 {
			return ErrAuditorSignaturesPresent
		}
		return nil
	}

	if len(ctx.TokenRequest.AuditorSignatures) == 0 {
		return ErrAuditorSignaturesMissing
	}

	auditors := ctx.PP.Auditors()
	for _, auditorSignature := range ctx.TokenRequest.AuditorSignatures {
		auditor := auditorSignature.Identity
		// check that issuer of this issue action is authorized
		if !slices.ContainsFunc(auditors, auditorSignature.Identity.Equal) {
			return errors.Errorf("auditor [%s] is not in auditors", auditor)
		}

		verifier, err := ctx.Deserializer.GetAuditorVerifier(c, auditor)
		if err != nil {
			return errors.Wrapf(err, "failed to deserialize auditor's public key")
		}
		_, err = ctx.SignatureProvider.HasBeenSignedBy(c, auditor, verifier)
		if err != nil {
			return errors.Wrap(err, "failed to verify auditor's signature")
		}
	}
	return nil
}
