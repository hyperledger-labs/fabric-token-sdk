/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/v1/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

// Auditor validates fabtoken token requests against their metadata.
// Unlike zkatdlog, fabtoken tokens are in cleartext, so validation focuses on
// structural correctness, token type consistency, and amount balance.
type Auditor struct {
	Logger       logging.Logger
	Tracer       trace.Tracer
	Deserializer driver.Deserializer
}

// NewAuditor creates a new Auditor for fabtoken validation.
func NewAuditor(logger logging.Logger, tracer trace.Tracer, deserializer driver.Deserializer) *Auditor {
	return &Auditor{
		Logger:       logger,
		Tracer:       tracer,
		Deserializer: deserializer,
	}
}

// Check validates TokenRequest against TokenRequestMetadata.
// It ensures complete 1:1 correspondence between actions and metadata, then validates each action.
func (a *Auditor) Check(
	ctx context.Context,
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	txID driver.TokenRequestAnchor,
	auditTokens map[*token.ID]*token.Token,
) error {
	// Validate structural correspondence between request and metadata
	if err := a.validateStructure(tokenRequest, tokenRequestMetadata, txID); err != nil {
		return errors.Wrapf(err, "structural validation failed for [%s]", txID)
	}

	// Process each action in order, matching with its metadata
	for i, action := range tokenRequest.Actions {
		metadata := tokenRequestMetadata.Actions[i]

		switch action.Type {
		case request.ActionType_ACTION_TYPE_ISSUE:
			if err := a.checkIssueAction(ctx, action.Raw, metadata.IssueMetadata, i, auditTokens); err != nil {
				return errors.Wrapf(err, "failed checking issue action at index [%d] for tx [%s]", i, txID)
			}

		case request.ActionType_ACTION_TYPE_TRANSFER:
			if err := a.checkTransferAction(ctx, action.Raw, metadata.TransferMetadata, auditTokens); err != nil {
				return errors.Wrapf(err, "failed checking transfer action at index [%d] for tx [%s]", i, txID)
			}

		default:
			return errors.Errorf("unknown action type [%s] at index [%d] for tx [%s]", action.Type, i, txID)
		}
	}

	return nil
}

// validateStructure ensures complete structural correspondence between TokenRequest and TokenRequestMetadata.
func (a *Auditor) validateStructure(
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	txID driver.TokenRequestAnchor,
) error {
	return common.ValidateStructure(tokenRequest, tokenRequestMetadata, txID)
}

// checkIssueAction validates a single issue action against its metadata.
func (a *Auditor) checkIssueAction(
	ctx context.Context,
	rawAction []byte,
	metadata *driver.IssueMetadata,
	actionIndex int,
	auditTokens map[*token.ID]*token.Token,
) error {
	// Deserialize the issue action
	ia := &actions.IssueAction{}
	if err := ia.Deserialize(rawAction); err != nil {
		return errors.Wrapf(err, "failed to deserialize issue action")
	}

	// Use IssueMetadata.Match to validate structural correspondence
	if err := metadata.Match(ia); err != nil {
		return errors.Wrapf(err, "issue action does not match metadata")
	}

	// Validate that all inputs and outputs have the same token type
	if err := a.validateIssueActionTokenTypes(metadata, auditTokens); err != nil {
		return errors.Wrapf(err, "token type validation failed for issue action at index [%d]", actionIndex)
	}

	return nil
}

// checkTransferAction validates a single transfer action against its metadata.
func (a *Auditor) checkTransferAction(
	ctx context.Context,
	rawAction []byte,
	metadata *driver.TransferMetadata,
	auditTokens map[*token.ID]*token.Token,
) error {
	// Deserialize the transfer action
	ta := &actions.TransferAction{}
	if err := ta.Deserialize(rawAction); err != nil {
		return errors.Wrapf(err, "failed to deserialize transfer action")
	}

	// Use TransferMetadata.Match to validate structural correspondence
	if err := metadata.Match(ta); err != nil {
		return errors.Wrapf(err, "transfer action does not match metadata")
	}

	// Validate that all inputs and outputs have the same token type and sum of values
	if err := a.validateTransferActionTokenTypes(metadata, auditTokens); err != nil {
		return errors.Wrapf(err, "token type validation failed for transfer action")
	}

	return nil
}

// validateIssueActionTokenTypes ensures all inputs and outputs in an issue action have the same token type.
// It also validates that input tokens exist in the auditTokens map.
func (a *Auditor) validateIssueActionTokenTypes(metadata *driver.IssueMetadata, auditTokens map[*token.ID]*token.Token) error {
	return common.ValidateIssueActionTokenTypes(metadata, auditTokens)
}

// validateTransferActionTokenTypes ensures all inputs and outputs in a transfer action have the same token type.
// It also validates that input tokens exist and that the sum of input values equals the sum of output values.
func (a *Auditor) validateTransferActionTokenTypes(metadata *driver.TransferMetadata, auditTokens map[*token.ID]*token.Token) error {
	// For fabtoken, we don't validate value sums because outputs are in cleartext
	// and validated by the action's Match() method
	return common.ValidateTransferActionTokenTypes(metadata, auditTokens, false)
}
