/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

func AuditingSignaturesValidate[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](ctx *Context[P, T, TA, IA, DS]) error {
	if len(ctx.PP.Auditors()) == 0 {
		// enforce no auditor signatures are attached
		if len(ctx.TokenRequest.AuditorSignatures) != 0 {
			return errors.New("auditor signatures are not empty")
		}
	}

	for _, auditorSignature := range ctx.TokenRequest.AuditorSignatures {
		auditor := auditorSignature.Identity
		verifier, err := ctx.Deserializer.GetAuditorVerifier(auditor)
		if err != nil {
			return errors.Errorf("failed to deserialize auditor's public key")
		}
		_, err = ctx.SignatureProvider.HasBeenSignedBy(auditor, verifier)
		if err != nil {
			return errors.Errorf("failed to verify auditor's signature")
		}
	}
	return nil

}
