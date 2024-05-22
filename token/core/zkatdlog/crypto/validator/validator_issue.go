/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

func IssueValidate(ctx *Context) error {
	action := ctx.IssueAction

	commitments, err := action.GetCommitments()
	if err != nil {
		return errors.New("failed to verify issue")
	}
	if err := issue.NewVerifier(
		commitments,
		ctx.PP).Verify(action.GetProof()); err != nil {
		return err
	}

	issuers := ctx.PP.Issuers
	if len(issuers) != 0 {
		// Check the issuer is among those known
		found := false
		for _, issuer := range issuers {
			if bytes.Equal(action.Issuer, issuer) {
				found = true
				break
			}
		}
		if !found {
			return errors.Errorf("issuer [%s] is not in issuers", driver.Identity(action.Issuer).String())
		}
	}

	verifier, err := ctx.Deserializer.GetIssuerVerifier(action.Issuer)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for [%s]", driver.Identity(action.Issuer).String())
	}
	if _, err := ctx.SignatureProvider.HasBeenSignedBy(action.Issuer, verifier); err != nil {
		return errors.Wrapf(err, "failed verifying signature")
	}
	return nil
}
