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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/v1/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
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
}

// NewAuditor creates a new Auditor.
func NewAuditor(logger logging.Logger, tracer trace.Tracer, infoMatcher InfoMatcher, pp []*math.G1, c *math.Curve) *Auditor {
	return &Auditor{
		Logger:         logger,
		Tracer:         tracer,
		InfoMatcher:    infoMatcher,
		PedersenParams: pp,
		Curve:          c,
	}
}

// Check validates TokenRequest against TokenRequestMetadata.
// It ensures complete 1:1 correspondence between actions and metadata, then validates each action.
func (a *Auditor) Check(
	ctx context.Context,
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	inputTokens [][]*token.Token,
	txID driver.TokenRequestAnchor,
) error {
	// Step 1: Validate structural correspondence between request and metadata
	if err := a.validateStructure(tokenRequest, tokenRequestMetadata, txID); err != nil {
		return errors.Wrapf(err, "structural validation failed for [%s]", txID)
	}

	// Step 2: Process each action in order, matching with its metadata
	transferIndex := 0
	for i, action := range tokenRequest.Actions {
		metadata := tokenRequestMetadata.Actions[i]

		// Verify ActionID matches position
		if metadata.ActionID != uint32(i) {
			return errors.Errorf("action at index [%d] has mismatched ActionID [%d] for tx [%s]", i, metadata.ActionID, txID)
		}

		// Process based on action type
		switch action.Type {
		case request.ActionType_ACTION_TYPE_ISSUE:
			if metadata.IssueMetadata == nil {
				return errors.Errorf("action at index [%d] is ISSUE but metadata has no IssueMetadata for tx [%s]", i, txID)
			}
			if err := a.checkIssueAction(ctx, action.Raw, metadata.IssueMetadata, i); err != nil {
				return errors.Wrapf(err, "failed checking issue action at index [%d] for tx [%s]", i, txID)
			}

		case request.ActionType_ACTION_TYPE_TRANSFER:
			if metadata.TransferMetadata == nil {
				return errors.Errorf("action at index [%d] is TRANSFER but metadata has no TransferMetadata for tx [%s]", i, txID)
			}
			// Get input tokens for this transfer
			var transferInputs []*token.Token
			if transferIndex < len(inputTokens) {
				transferInputs = inputTokens[transferIndex]
			}
			if err := a.checkTransferAction(ctx, action.Raw, metadata.TransferMetadata, transferInputs); err != nil {
				return errors.Wrapf(err, "failed checking transfer action at index [%d] for tx [%s]", i, txID)
			}
			transferIndex++

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
	// Validate action count matches metadata count
	if len(tokenRequest.Actions) != len(tokenRequestMetadata.Actions) {
		return errors.Errorf(
			"action count mismatch: request has [%d] actions but metadata has [%d] actions for tx [%s]",
			len(tokenRequest.Actions),
			len(tokenRequestMetadata.Actions),
			txID,
		)
	}

	// Validate each action has corresponding metadata with correct type
	for i, action := range tokenRequest.Actions {
		if action == nil {
			return errors.Errorf("action at index [%d] is nil for tx [%s]", i, txID)
		}

		metadata := tokenRequestMetadata.Actions[i]
		if metadata == nil {
			return errors.Errorf("metadata at index [%d] is nil for tx [%s]", i, txID)
		}

		// Verify ActionID matches position
		if metadata.ActionID != uint32(i) {
			return errors.Errorf(
				"metadata at index [%d] has incorrect ActionID [%d] for tx [%s]",
				i,
				metadata.ActionID,
				txID,
			)
		}

		// Verify action type matches metadata type
		switch action.Type {
		case request.ActionType_ACTION_TYPE_ISSUE:
			if metadata.IssueMetadata == nil {
				return errors.Errorf(
					"action at index [%d] is ISSUE but metadata has no IssueMetadata for tx [%s]",
					i,
					txID,
				)
			}
			if metadata.TransferMetadata != nil {
				return errors.Errorf(
					"action at index [%d] is ISSUE but metadata also has TransferMetadata for tx [%s]",
					i,
					txID,
				)
			}

		case request.ActionType_ACTION_TYPE_TRANSFER:
			if metadata.TransferMetadata == nil {
				return errors.Errorf(
					"action at index [%d] is TRANSFER but metadata has no TransferMetadata for tx [%s]",
					i,
					txID,
				)
			}
			if metadata.IssueMetadata != nil {
				return errors.Errorf(
					"action at index [%d] is TRANSFER but metadata also has IssueMetadata for tx [%s]",
					i,
					txID,
				)
			}

		default:
			return errors.Errorf(
				"action at index [%d] has unknown type [%s] for tx [%s]",
				i,
				action.Type,
				txID,
			)
		}
	}

	return nil
}

// checkIssueAction validates a single issue action against its metadata.
// It uses IssueMetadata.Match to validate structural correspondence, then validates token commitments and identities.
func (a *Auditor) checkIssueAction(
	ctx context.Context,
	rawAction []byte,
	metadata *driver.IssueMetadata,
	actionIndex int,
) error {
	// Deserialize the issue action
	ia := &issue.Action{}
	if err := ia.Deserialize(rawAction); err != nil {
		return errors.Wrapf(err, "failed to deserialize issue action")
	}

	// Use IssueMetadata.Match to validate structural correspondence
	// This validates all fields including issuer, inputs, outputs, extra signers, and proofs
	if err := metadata.Match(ia); err != nil {
		return errors.Wrapf(err, "issue action does not match metadata")
	}

	// Create inspectable tokens and validate commitments
	for i, output := range ia.Outputs {
		if output == nil {
			return errors.Errorf("output at index [%d] is nil", i)
		}

		outputMetadata := metadata.Outputs[i]
		if outputMetadata == nil {
			return errors.Errorf("output metadata at index [%d] is nil", i)
		}

		// Issue actions cannot redeem tokens
		if output.IsRedeem() {
			return errors.Errorf("issue action cannot redeem tokens")
		}

		// Validate receiver exists
		if len(outputMetadata.Receivers) == 0 || outputMetadata.Receivers[0] == nil {
			return errors.Errorf("output at index [%d] must have at least one receiver", i)
		}

		// Deserialize token metadata
		tokenMetadata := &token.Metadata{}
		if err := tokenMetadata.Deserialize(outputMetadata.OutputMetadata); err != nil {
			return errors.Wrapf(err, "failed to deserialize token metadata at index [%d]", i)
		}

		// Create inspectable token
		inspectable, err := NewInspectableToken(
			output,
			outputMetadata.Receivers[0].AuditInfo,
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

	// Validate issuer identity
	issuerIdentity := InspectableIdentity{
		Identity:         ia.Issuer,
		IdentityFromMeta: metadata.Issuer.Identity,
		AuditInfo:        metadata.Issuer.AuditInfo,
	}
	if err := a.InspectIdentity(ctx, a.InfoMatcher, &issuerIdentity, actionIndex); err != nil {
		return errors.Wrapf(err, "failed checking issuer identity")
	}

	return nil
}

// checkTransferAction validates a single transfer action against its metadata.
// It validates all fields correspond, then validates token commitments and identities.
func (a *Auditor) checkTransferAction(
	ctx context.Context,
	rawAction []byte,
	metadata *driver.TransferMetadata,
	inputTokens []*token.Token,
) error {
	// Deserialize the transfer action
	ta := &transfer.Action{}
	if err := ta.Deserialize(rawAction); err != nil {
		return errors.Wrapf(err, "failed to deserialize transfer action")
	}

	// Use TransferMetadata.Match to validate structural correspondence
	// This validates all fields including issuer, inputs, outputs, extra signers, and proofs
	if err := metadata.Match(ta); err != nil {
		return errors.Wrapf(err, "transfer action does not match metadata")
	}

	// Validate input tokens match metadata
	if len(inputTokens) != len(metadata.Inputs) {
		return errors.Errorf(
			"transfer has [%d] input tokens but metadata has [%d] inputs",
			len(inputTokens),
			len(metadata.Inputs),
		)
	}

	// Validate and inspect inputs
	for i, inputToken := range inputTokens {
		if inputToken == nil {
			return errors.Errorf("input token at index [%d] is nil", i)
		}

		inputMetadata := metadata.Inputs[i]
		if inputMetadata == nil || len(inputMetadata.Senders) == 0 || inputMetadata.Senders[0] == nil {
			return errors.Errorf("invalid metadata for input at index [%d]", i)
		}

		// Create inspectable token for input (only need audit info for sender)
		inspectable, err := NewInspectableToken(
			inputToken,
			inputMetadata.Senders[0].AuditInfo,
			"",  // Type not needed for input validation
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

	// Validate outputs match metadata
	if len(ta.Outputs) != len(metadata.Outputs) {
		return errors.Errorf(
			"transfer action has [%d] outputs but metadata has [%d] outputs",
			len(ta.Outputs),
			len(metadata.Outputs),
		)
	}

	// Validate and inspect outputs
	for i, output := range ta.Outputs {
		if output == nil {
			return errors.Errorf("output at index [%d] is nil", i)
		}

		outputMetadata := metadata.Outputs[i]
		if outputMetadata == nil {
			return errors.Errorf("output metadata at index [%d] is nil", i)
		}

		// Deserialize token metadata
		tokenMetadata := &token.Metadata{}
		if err := tokenMetadata.Deserialize(outputMetadata.OutputMetadata); err != nil {
			return errors.Wrapf(err, "failed to deserialize token metadata at index [%d]", i)
		}

		// Create inspectable token
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

// CheckTransferRequests verifies that the commitments in transfer inputs and outputs match the information provided in the clear.
func (a *Auditor) CheckTransferRequests(ctx context.Context, inputs [][]*InspectableToken, outputsFromTransfer [][]*InspectableToken, txID driver.TokenRequestAnchor) error {
	// Inspect outputs of each transfer action
	for k, transferred := range outputsFromTransfer {
		err := a.InspectOutputs(ctx, transferred)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

	// Inspect inputs of each transfer action
	for k, i := range inputs {
		err := a.InspectInputs(ctx, i)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

	return nil
}

// CheckIssueRequests verifies that the commitments in issue outputs match the information provided in the clear.
func (a *Auditor) CheckIssueRequests(ctx context.Context, outputsFromIssue [][]*InspectableToken, txID driver.TokenRequestAnchor) error {
	// Inspect outputs of each issue action
	for k, issued := range outputsFromIssue {
		err := a.InspectOutputs(ctx, issued)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th issue in tx [%s] failed", k, txID)
		}
	}

	return nil
}

// InspectOutputs verifies that the commitments in an array of outputs matches the information provided in the clear.
func (a *Auditor) InspectOutputs(ctx context.Context, tokens []*InspectableToken) error {
	for i, t := range tokens {
		err := a.InspectOutput(ctx, t, i)
		if err != nil {
			return errors.Wrapf(err, "failed inspecting output [%d]", i)
		}
	}

	return nil
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

// InspectInputs verifies that the commitment in an array of inputs matches the information provided in the clear.
func (a *Auditor) InspectInputs(ctx context.Context, inputs []*InspectableToken) error {
	for i, input := range inputs {
		if input == nil {
			return errors.Errorf("invalid input at index [%d]", i)
		}

		// Verify input owner identity
		if !input.Identity.Identity.IsNone() {
			if err := a.InspectIdentity(ctx, a.InfoMatcher, &input.Identity, i); err != nil {
				return errors.Wrapf(err, "failed inspecting input at index [%d]", i)
			}
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
			return errors.Errorf("failed to inspect identity at index [%d]: identity does not match the identity form metadata", index)
		}
	}
	// Use InfoMatcher to verify that AuditInfo corresponds to the Identity
	if err := matcher.MatchIdentity(ctx, identity.Identity, identity.AuditInfo); err != nil {
		return errors.Wrapf(err, "owner at index [%d] does not match the provided opening", index)
	}

	return nil
}

// GetAuditInfoForIssues returns an array of InspectableToken for each issue action.
// It takes an array of serialized issue actions and an array of issue metadata.
func (a *Auditor) GetAuditInfoForIssues(issues [][]byte, issueMetadata []*driver.IssueMetadata) ([][]*InspectableToken, []InspectableIdentity, error) {
	if len(issues) != len(issueMetadata) {
		return nil, nil, errors.Errorf("number of issues does not match number of provided metadata")
	}
	outputs := make([][]*InspectableToken, len(issues))
	identities := make([]InspectableIdentity, len(issues))
	for k, md := range issueMetadata {
		ia := &issue.Action{}
		err := ia.Deserialize(issues[k])
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to deserialize issue action at index [%d]", k)
		}

		if len(ia.Outputs) != len(md.Outputs) {
			return nil, nil, errors.Errorf("number of output does not match number of provided metadata")
		}
		outputs[k] = make([]*InspectableToken, len(md.Outputs))
		for i, o := range md.Outputs {
			if o == nil {
				return nil, nil, errors.Errorf("output at index [%d] is nil", i)
			}
			metadata := &token.Metadata{}
			err = metadata.Deserialize(o.OutputMetadata)
			if err != nil {
				return nil, nil, err
			}
			if ia.Outputs[i] == nil {
				return nil, nil, errors.Errorf("output token at index [%d] is nil", i)
			}
			// Issue actions cannot redeem tokens
			if ia.Outputs[i].IsRedeem() {
				return nil, nil, errors.Errorf("issue cannot redeem tokens")
			}
			if len(o.Receivers) == 0 || o.Receivers[0] == nil {
				return nil, nil, errors.Errorf("issue must have at least one receiver")
			}

			// Create auditable token using the metadata provided in the request
			outputs[k][i], err = NewInspectableToken(
				ia.Outputs[i],
				o.Receivers[0].AuditInfo,
				metadata.Type,
				metadata.Value,
				metadata.BlindingFactor,
			)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to create auditable token at index [%d]", i)
			}
		}
		identities[k] = InspectableIdentity{
			Identity:         ia.Issuer,
			IdentityFromMeta: md.Issuer.Identity,
			AuditInfo:        md.Issuer.AuditInfo,
		}
	}

	return outputs, identities, nil
}

// GetAuditInfoForTransfers returns an array of InspectableToken for each transfer action.
// It takes an array of serialized transfer actions, an array of transfer metadata and input tokens.
func (a *Auditor) GetAuditInfoForTransfers(transfers [][]byte, metadata []*driver.TransferMetadata, inputs [][]*token.Token) ([][]*InspectableToken, [][]*InspectableToken, error) {
	if len(transfers) != len(metadata) {
		return nil, nil, errors.Errorf("number of transfers does not match the number of provided metadata")
	}
	if len(inputs) != len(metadata) {
		return nil, nil, errors.Errorf("number of inputs does not match the number of provided metadata")
	}
	auditableInputs := make([][]*InspectableToken, len(inputs))
	outputs := make([][]*InspectableToken, len(transfers))
	for k, transferMetadata := range metadata {
		if len(transferMetadata.Inputs) != len(inputs[k]) {
			return nil, nil, errors.Errorf("number of inputs does not match the number of senders [%d]!=[%d]", len(transferMetadata.Inputs), len(inputs[k]))
		}
		// Process auditable inputs
		auditableInputs[k] = make([]*InspectableToken, len(transferMetadata.Inputs))
		for i := range len(transferMetadata.Inputs) {
			var err error
			if inputs[k][i] == nil {
				return nil, nil, errors.Errorf("input[%d][%d] is nil", k, i)
			}
			if transferMetadata.Inputs[i] == nil || len(transferMetadata.Inputs[i].Senders) == 0 || transferMetadata.Inputs[i].Senders[0] == nil {
				return nil, nil, errors.Errorf("invalid metadata for input[%d][%d]", k, i)
			}
			// For inputs, we only need the audit info to identify the sender
			auditableInputs[k][i], err = NewInspectableToken(inputs[k][i], transferMetadata.Inputs[i].Senders[0].AuditInfo, "", nil, nil)
			if err != nil {
				return nil, nil, err
			}
		}
		ta := &transfer.Action{}
		err := ta.Deserialize(transfers[k])
		if err != nil {
			return nil, nil, err
		}
		if len(ta.Outputs) != len(transferMetadata.Outputs) {
			return nil, nil, errors.Errorf("number of outputs does not match the number of output metadata [%d]!=[%d]", len(ta.Outputs), len(transferMetadata.Outputs))
		}
		// Process auditable outputs
		outputs[k] = make([]*InspectableToken, len(ta.Outputs))
		for i := range len(ta.Outputs) {
			if ta.Outputs[i] == nil {
				return nil, nil, errors.Errorf("output token at index [%d] is nil", i)
			}

			if transferMetadata.Outputs[i] == nil {
				return nil, nil, errors.Errorf("metadata for output token at index [%d] is nil", i)
			}
			ti := &token.Metadata{}
			err = ti.Deserialize(transferMetadata.Outputs[i].OutputMetadata)
			if err != nil {
				return nil, nil, err
			}
			// TODO: we need to check also how many recipients the output contains, and check them all in isolation and compatibility
			outputs[k][i], err = NewInspectableToken(
				ta.Outputs[i],
				transferMetadata.Outputs[i].OutputAuditInfo,
				ti.Type,
				ti.Value,
				ti.BlindingFactor,
			)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return auditableInputs, outputs, nil
}

// commit computes a Pedersen commitment for the given vector and generators.
func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) *math.G1 {
	com := c.NewG1()
	for i := range vector {
		com.Add(generators[i].Mul(vector[i]))
	}

	return com
}
