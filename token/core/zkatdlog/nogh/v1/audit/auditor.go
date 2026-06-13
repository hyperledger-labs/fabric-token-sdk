/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"bytes"
	"context"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/v1/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

const (
	// tokenTypeNotNeeded is used when creating inspectable tokens for validation
	// where the token type is not required (e.g., for transfer inputs)
	tokenTypeNotNeeded token2.Type = ""
)

// SigningIdentity is an alias for driver.SigningIdentity
//
//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity
type SigningIdentity = driver.SigningIdentity

// InfoMatcher deserialize audit information
//
//go:generate counterfeiter -o mock/info_matcher.go -fake-name InfoMatcher . InfoMatcher
type InfoMatcher interface {
	MatchIdentity(ctx context.Context, id driver.Identity, ai []byte) error
	Recipients(raw driver.Identity) ([]driver.Identity, error)
}

// InspectableIdentity contains the identity and its corresponding audit info.
type InspectableIdentity struct {
	Identity         driver.Identity
	IdentityFromMeta driver.Identity
	AuditInfo        []byte
}

// InspectableData contains the token data and its opening.
type InspectableData struct {
	Data      *math.G1
	TokenType token2.Type
	Value     *math.Zr
	BF        *math.Zr
}

// InspectableToken contains a zkat token and the information that allows
// an auditor to learn its content.
type InspectableToken struct {
	Identity InspectableIdentity
	Data     InspectableData
}

// NewInspectableToken creates a new InspectableToken.
func NewInspectableToken(
	token *token.Token,
	ownerAuditInfo []byte,
	tokenType token2.Type,
	value *math.Zr,
	bf *math.Zr,
) (*InspectableToken, error) {
	return &InspectableToken{
		Identity: InspectableIdentity{
			Identity:  token.Owner,
			AuditInfo: ownerAuditInfo,
		},
		Data: InspectableData{
			Data:      token.Data,
			TokenType: tokenType,
			Value:     value,
			BF:        bf,
		},
	}, nil
}

// Auditor inspects zkat tokens and their owners.
type Auditor struct {
	Logger logging.Logger
	Tracer trace.Tracer
	// Owner Identity InfoMatcher
	InfoMatcher InfoMatcher
	// Pedersen generators used to compute TokenData
	PedersenParams []*math.G1
	// Elliptic curve
	Curve *math.Curve
	// Precision for token quantities
	Precision uint64
}

// NewAuditor creates a new Auditor.
func NewAuditor(logger logging.Logger, tracer trace.Tracer, infoMatcher InfoMatcher, pp []*math.G1, c *math.Curve, precision uint64) *Auditor {
	return &Auditor{
		Logger:         logger,
		Tracer:         tracer,
		InfoMatcher:    infoMatcher,
		PedersenParams: pp,
		Curve:          c,
		Precision:      precision,
	}
}

// Check validates TokenRequest against TokenRequestMetadata.
// It ensures complete 1:1 correspondence between actions and metadata, then validates each action.
//
// Note: TokenRequestMetadata.Application field is not validated by this auditor.
// Application-specific metadata should be validated by higher-level application logic if needed.
func (a *Auditor) Check(
	ctx context.Context,
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	txID driver.TokenRequestAnchor,
	auditTokens map[*token2.ID]*token2.Token,
) error {
	if tokenRequest == nil {
		return errors.Errorf("tokenRequest cannot be nil for tx [%s]", txID)
	}
	if tokenRequestMetadata == nil {
		return errors.Errorf("tokenRequestMetadata cannot be nil for tx [%s]", txID)
	}
	if auditTokens == nil {
		return errors.Errorf("auditTokens cannot be nil for tx [%s]", txID)
	}

	// Validate structural correspondence between request and metadata
	// This includes action counts, ActionIDs, and metadata type alignment
	if err := a.validateStructure(tokenRequest, tokenRequestMetadata, txID); err != nil {
		return errors.Wrapf(err, "structural validation failed for [%s]", txID)
	}

	// Process each action in order, matching with its metadata
	// validateStructure() has already confirmed metadata types are correct
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
// It validates that action counts match, ActionIDs are correct, and action types align with metadata types.
func (a *Auditor) validateStructure(
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	txID driver.TokenRequestAnchor,
) error {
	return common.ValidateStructure(tokenRequest, tokenRequestMetadata, txID)
}

// checkIssueAction validates a single issue action against its metadata.
// It uses IssueMetadata.Match to validate structural correspondence (counts, types),
// then validates semantic correctness (token commitments and identities).
func (a *Auditor) checkIssueAction(
	ctx context.Context,
	rawAction []byte,
	metadata *driver.IssueMetadata,
	actionIndex int,
	auditTokens map[*token2.ID]*token2.Token,
) error {
	// Deserialize the issue action
	ia := &issue.Action{}
	if err := ia.Deserialize(rawAction); err != nil {
		return errors.Wrapf(err, "failed to deserialize issue action")
	}

	// Use IssueMetadata.Match to validate structural correspondence
	// This validates counts and types but not semantic correctness
	if err := metadata.Match(ia); err != nil {
		return errors.Wrapf(err, "issue action does not match metadata")
	}

	// Validate inputs (for token upgrades)
	if err := a.validateIssueInputs(ia.Inputs, metadata.Inputs); err != nil {
		return err
	}

	// Validate outputs
	if err := a.validateIssueOutputs(ctx, ia.Outputs, metadata.Outputs); err != nil {
		return err
	}

	// Validate that all inputs and outputs have the same token type
	if err := a.validateIssueActionTokenTypes(metadata, auditTokens); err != nil {
		return errors.Wrapf(err, "token type validation failed for issue action at index [%d]", actionIndex)
	}

	// Validate issuer identity
	if err := a.validateIssuer(ctx, ia.Issuer, &metadata.Issuer, actionIndex); err != nil {
		return err
	}

	return nil
}

// validateIssueInputs validates issue action inputs against metadata.
func (a *Auditor) validateIssueInputs(inputs []*issue.ActionInput, inputsMetadata []*driver.IssueInputMetadata) error {
	for i, input := range inputs {
		if input == nil {
			return errors.Errorf("input at index [%d] is nil", i)
		}
		if input.Token == nil {
			return errors.Errorf("input token at index [%d] is nil", i)
		}

		inputMetadata := inputsMetadata[i]
		if inputMetadata == nil {
			return errors.Errorf("input metadata at index [%d] is nil", i)
		}

		// Validate TokenID matches the actual input token ID
		if inputMetadata.TokenID != nil {
			if !inputMetadata.TokenID.Equal(input.ID) {
				return errors.Errorf(
					"token ID mismatch at input [%d]: metadata has [%s:%d] but action has [%s:%d]",
					i,
					inputMetadata.TokenID.TxId, inputMetadata.TokenID.Index,
					input.ID.TxId, input.ID.Index,
				)
			}
		}
	}

	return nil
}

// validateIssueOutputs validates issue action outputs against metadata.
func (a *Auditor) validateIssueOutputs(ctx context.Context, outputs []*token.Token, outputsMetadata []*driver.IssueOutputMetadata) error {
	for i, output := range outputs {
		if output == nil {
			return errors.Errorf("output at index [%d] is nil", i)
		}

		outputMetadata := outputsMetadata[i]
		if outputMetadata == nil {
			return errors.Errorf("output metadata at index [%d] is nil", i)
		}

		// Issue actions cannot redeem tokens
		if output.IsRedeem() {
			return errors.Errorf("issue action cannot redeem tokens")
		}

		// Deserialize token metadata
		tokenMetadata := &token.Metadata{}
		if err := tokenMetadata.Deserialize(outputMetadata.OutputMetadata); err != nil {
			return errors.Wrapf(err, "failed to deserialize token metadata at index [%d]", i)
		}

		// Create inspectable token and validate commitment
		inspectable, err := NewInspectableToken(
			output,
			outputMetadata.OutputAuditInfo,
			tokenMetadata.Type,
			tokenMetadata.Value,
			tokenMetadata.BlindingFactor,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to create inspectable token for output [%d]", i)
		}

		if err := a.InspectOutput(ctx, inspectable, i); err != nil {
			return errors.Wrapf(err, "failed inspecting output [%d]", i)
		}

		// Validate receivers match recipients
		if err := a.validateOutputReceivers(ctx, output.Owner, outputMetadata.Receivers, i, true); err != nil {
			return err
		}
	}

	return nil
}

// validateIssuer validates the issuer identity against metadata.
func (a *Auditor) validateIssuer(ctx context.Context, issuer driver.Identity, issuerMetadata *driver.AuditableIdentity, actionIndex int) error {
	issuerIdentity := InspectableIdentity{
		Identity:         issuer,
		IdentityFromMeta: issuerMetadata.Identity,
		AuditInfo:        issuerMetadata.AuditInfo,
	}
	if err := a.InspectIdentity(ctx, a.InfoMatcher, &issuerIdentity, actionIndex); err != nil {
		return errors.Wrapf(err, "failed checking issuer identity")
	}

	return nil
}

// checkTransferAction validates a single transfer action against its metadata.
// It uses TransferMetadata.Match to validate structural correspondence (counts, types),
// then validates semantic correctness (token commitments and identities).
func (a *Auditor) checkTransferAction(
	ctx context.Context,
	rawAction []byte,
	metadata *driver.TransferMetadata,
	auditTokens map[*token2.ID]*token2.Token,
) error {
	// Deserialize the transfer action
	ta := &transfer.Action{}
	if err := ta.Deserialize(rawAction); err != nil {
		return errors.Wrapf(err, "failed to deserialize transfer action")
	}

	// Use TransferMetadata.Match to validate structural correspondence
	// This validates counts and types but not semantic correctness
	if err := metadata.Match(ta); err != nil {
		return errors.Wrapf(err, "transfer action does not match metadata")
	}

	// Validate inputs
	if err := a.validateTransferInputs(ctx, ta.Inputs, metadata.Inputs); err != nil {
		return err
	}

	// Validate outputs
	if err := a.validateTransferOutputs(ctx, ta.Outputs, metadata.Outputs); err != nil {
		return err
	}

	// Validate that all inputs and outputs have the same token type and sum of values
	if err := a.validateTransferActionTokenTypes(metadata, auditTokens); err != nil {
		return errors.Wrapf(err, "token type validation failed for transfer action")
	}

	return nil
}

// validateTransferInputs validates transfer action inputs against metadata.
func (a *Auditor) validateTransferInputs(ctx context.Context, inputs []*transfer.ActionInput, inputsMetadata []*driver.TransferInputMetadata) error {
	for i, actionInput := range inputs {
		if actionInput == nil || actionInput.Token == nil {
			return errors.Errorf("input at index [%d] is nil", i)
		}
		if actionInput.ID == nil {
			return errors.Errorf("input at index [%d] has nil ID", i)
		}

		inputMetadata := inputsMetadata[i]
		if inputMetadata == nil {
			return errors.Errorf("metadata for input at index [%d] is nil", i)
		}

		// Validate exactly one sender in metadata
		if len(inputMetadata.Senders) != 1 {
			return errors.Errorf(
				"input metadata at index [%d] must have exactly one sender, found [%d]",
				i,
				len(inputMetadata.Senders),
			)
		}

		if inputMetadata.Senders[0] == nil {
			return errors.Errorf("sender at index [%d] is nil", i)
		}

		// Validate sender identity matches token owner
		if !bytes.Equal(inputMetadata.Senders[0].Identity, actionInput.Token.Owner) {
			return errors.Errorf(
				"sender identity at index [%d] does not match token owner",
				i,
			)
		}

		// Validate TokenID matches the actual input token ID
		if inputMetadata.TokenID != nil && !inputMetadata.TokenID.Equal(*actionInput.ID) {
			return errors.Errorf(
				"token ID mismatch at input [%d]: metadata has [%s:%d] but action has [%s:%d]",
				i,
				inputMetadata.TokenID.TxId, inputMetadata.TokenID.Index,
				actionInput.ID.TxId, actionInput.ID.Index,
			)
		}

		// Create inspectable token for input (only need audit info for sender)
		inspectable, err := NewInspectableToken(
			actionInput.Token,
			inputMetadata.Senders[0].AuditInfo,
			tokenTypeNotNeeded,
			nil, // Value not needed for input validation
			nil, // BlindingFactor not needed for input validation
		)
		if err != nil {
			return errors.Wrapf(err, "failed to create inspectable token for input at index [%d]", i)
		}

		// Validate sender identity
		if !inspectable.Identity.Identity.IsNone() {
			if err := a.InspectIdentity(ctx, a.InfoMatcher, &inspectable.Identity, i); err != nil {
				return errors.Wrapf(err, "failed inspecting input sender at index [%d]", i)
			}
		}
	}

	return nil
}

// validateTransferOutputs validates transfer action outputs against metadata.
func (a *Auditor) validateTransferOutputs(ctx context.Context, outputs []*token.Token, outputsMetadata []*driver.TransferOutputMetadata) error {
	for i, output := range outputs {
		if output == nil {
			return errors.Errorf("output at index [%d] is nil", i)
		}

		outputMetadata := outputsMetadata[i]
		if outputMetadata == nil {
			return errors.Errorf("output metadata at index [%d] is nil", i)
		}

		// Validate receivers for non-redeem outputs
		if !output.IsRedeem() {
			if err := a.validateOutputReceivers(ctx, output.Owner, outputMetadata.Receivers, i, false); err != nil {
				return err
			}
		}

		// Deserialize token metadata
		tokenMetadata := &token.Metadata{}
		if err := tokenMetadata.Deserialize(outputMetadata.OutputMetadata); err != nil {
			return errors.Wrapf(err, "failed to deserialize token metadata at index [%d]", i)
		}

		// Create inspectable token using OutputAuditInfo (primary audit info for transfer outputs)
		inspectable, err := NewInspectableToken(
			output,
			outputMetadata.OutputAuditInfo,
			tokenMetadata.Type,
			tokenMetadata.Value,
			tokenMetadata.BlindingFactor,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to create inspectable token at index [%d]", i)
		}

		// Validate output commitment and identity
		if err := a.InspectOutput(ctx, inspectable, i); err != nil {
			return errors.Wrapf(err, "failed inspecting output at index [%d]", i)
		}
	}

	return nil
}

// validateOutputReceivers validates that output receivers in metadata match recipients extracted from the owner.
// The requireNonEmpty parameter controls whether empty receiver identities are allowed (issue vs transfer).
func (a *Auditor) validateOutputReceivers(
	ctx context.Context,
	owner driver.Identity,
	receivers []*driver.AuditableIdentity,
	outputIndex int,
	requireNonEmpty bool,
) error {
	// Extract recipients from the output owner
	recipients, err := a.InfoMatcher.Recipients(owner)
	if err != nil {
		return errors.Wrapf(err, "failed to extract recipients from output owner at index [%d]", outputIndex)
	}

	// Validate that recipients slice is non-empty
	if len(recipients) == 0 {
		return errors.Errorf("output at index [%d] has no recipients", outputIndex)
	}

	// Validate that metadata provides receivers for all recipients in the action
	if len(receivers) != len(recipients) {
		return errors.Errorf(
			"output at index [%d] has [%d] recipients but metadata has [%d] receivers",
			outputIndex, len(recipients), len(receivers),
		)
	}

	// Validate all receivers and their identities
	for j, receiver := range receivers {
		if receiver == nil {
			return errors.Errorf("receiver at index [%d] for output [%d] is nil", j, outputIndex)
		}

		// Validate that the recipient from owner matches the receiver identity in metadata
		// For issue actions, identity must be non-empty and match
		// For transfer actions, empty identity is allowed (optional validation)
		if requireNonEmpty {
			if len(receiver.Identity) == 0 || !bytes.Equal(recipients[j], receiver.Identity) {
				return errors.Errorf(
					"recipient at index [%d] for output [%d] does not match receiver identity in metadata",
					j, outputIndex,
				)
			}
		} else {
			if len(receiver.Identity) > 0 && !bytes.Equal(recipients[j], receiver.Identity) {
				return errors.Errorf(
					"recipient at index [%d] for output [%d] does not match receiver identity in metadata",
					j, outputIndex,
				)
			}
		}

		// Inspect receiver identity
		receiverInspectable := InspectableIdentity{
			Identity:  receiver.Identity,
			AuditInfo: receiver.AuditInfo,
		}
		if err := a.InspectIdentity(ctx, a.InfoMatcher, &receiverInspectable, outputIndex); err != nil {
			return errors.Wrapf(err, "failed inspecting receiver [%d] at output [%d]", j, outputIndex)
		}
	}

	return nil
}

// validateIssueActionTokenTypes ensures all inputs and outputs in an issue action have the same token type.
// It also validates that input tokens exist in the auditTokens map.
func (a *Auditor) validateIssueActionTokenTypes(metadata *driver.IssueMetadata, auditTokens map[*token2.ID]*token2.Token) error {
	return common.ValidateIssueActionTokenTypes(metadata, auditTokens)
}

// validateTransferActionTokenTypes ensures all inputs and outputs in a transfer action have the same token type.
// It also validates that input tokens exist and that the sum of input values equals the sum of output values
// (only when audit tokens are provided).
func (a *Auditor) validateTransferActionTokenTypes(metadata *driver.TransferMetadata, auditTokens map[*token2.ID]*token2.Token) error {
	return common.ValidateTransferActionTokenTypes(metadata, auditTokens, true, a.Precision)
}

// InspectOutput verifies that the commitments in an output token of a given index
// match the information provided in the clear.
func (a *Auditor) InspectOutput(ctx context.Context, output *InspectableToken, index int) error {
	if output == nil || output.Data.Data == nil {
		return errors.Errorf("invalid output at index [%d]", index)
	}
	// Recompute commitment from provided cleartext data
	tokenComm := commit([]*math.Zr{
		a.Curve.HashToZr([]byte(output.Data.TokenType)),
		output.Data.Value,
		output.Data.BF,
	}, a.PedersenParams, a.Curve)
	// Verify it matches the commitment in the token
	if !tokenComm.Equals(output.Data.Data) {
		return errors.Errorf("output at index [%d] does not match the provided opening", index)
	}
	// Verify owner identity if it's not a redeemed output
	if !output.Identity.Identity.IsNone() { // this is not a redeemed output
		if err := a.InspectIdentity(ctx, a.InfoMatcher, &output.Identity, index); err != nil {
			return errors.Wrapf(err, "failed inspecting output at index [%d]", index)
		}
	}

	return nil
}

// InspectIdentity verifies that the audit info matches the token owner.
func (a *Auditor) InspectIdentity(ctx context.Context, matcher InfoMatcher, identity *InspectableIdentity, index int) error {
	if identity.Identity.IsNone() {
		return errors.Errorf("identity at index [%d] is nil, cannot inspect it", index)
	}
	if len(identity.AuditInfo) == 0 {
		return errors.Errorf("failed to inspect identity at index [%d]: audit info is nil", index)
	}
	// If identity is provided in metadata, it must match the one in the action
	if len(identity.IdentityFromMeta) != 0 {
		// enforce equality
		if !bytes.Equal(identity.IdentityFromMeta, identity.Identity) {
			return errors.Errorf("failed to inspect identity at index [%d]: identity does not match the identity from metadata", index)
		}
	}
	// Use InfoMatcher to verify that AuditInfo corresponds to the Identity
	if err := matcher.MatchIdentity(ctx, identity.Identity, identity.AuditInfo); err != nil {
		return errors.Wrapf(err, "owner at index [%d] does not match the provided opening", index)
	}

	return nil
}

// commit computes a Pedersen commitment for the given vector and generators.
func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) *math.G1 {
	com := c.NewG1()
	for i := range vector {
		com.Add(generators[i].Mul(vector[i]))
	}

	return com
}
