/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"context"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

// IssueValidate validates the issue action
func IssueValidate(c context.Context, ctx *Context) error {
	action := ctx.IssueAction

	if err := action.Validate(); err != nil {
		return errors.Wrapf(err, "failed validating issue action")
	}

	commitments, err := action.GetCommitments()
	if err != nil {
		return errors.New("failed to verify issue")
	}
	if err := issue.NewVerifier(
		commitments,
		ctx.PP).Verify(action.GetProof()); err != nil {
		return err
	}

	// Check the issuer is among those known
	if issuers := ctx.PP.Issuers(); len(issuers) != 0 && !slices.ContainsFunc(issuers, action.Issuer.Equal) {
		return errors.Errorf("issuer [%s] is not in issuers", action.Issuer)
	}
	logger.Debugf("Found issue owner [%s]", action.Issuer)

	verifier, err := ctx.Deserializer.GetIssuerVerifier(c, action.Issuer)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for issuer [%s]", action.Issuer.String())
	}
	if _, err := ctx.SignatureProvider.HasBeenSignedBy(c, action.Issuer, verifier); err != nil {
		return errors.Wrapf(err, "failed verifying signature")
	}

	return nil
}
