/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"slices"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

func AuditingSignaturesValidate[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](ctx *Context[P, T, TA, IA, DS]) error {
	if len(ctx.PP.Auditors()) == 0 {
		// enforce no auditor signatures are attached
		if len(ctx.TokenRequest.AuditorSignatures) != 0 {
			return errors.New("auditor signatures are not empty")
		}
		return nil
	}

	auditors := ctx.PP.Auditors()
	for _, auditorSignature := range ctx.TokenRequest.AuditorSignatures {
		auditor := auditorSignature.Identity
		// check that issuer of this issue action is authorized
		if !slices.ContainsFunc(auditors, auditorSignature.Identity.Equal) {
			return errors.Errorf("auditor [%s] is not in auditors", auditor)
		}

		verifier, err := ctx.Deserializer.GetAuditorVerifier(auditor)
		if err != nil {
			return errors.Wrapf(err, "failed to deserialize auditor's public key")
		}
		_, err = ctx.SignatureProvider.HasBeenSignedBy(auditor, verifier)
		if err != nil {
			return errors.Wrap(err, "failed to verify auditor's signature")
		}
	}
	return nil
}
