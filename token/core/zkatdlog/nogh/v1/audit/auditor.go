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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// SigningIdentity models a signing identity
type SigningIdentity interface {
	driver.SigningIdentity
}

// InfoMatcher deserialize audit information
type InfoMatcher interface {
	MatchIdentity(ctx context.Context, id driver.Identity, ai []byte) error
}

type InspectableIdentity struct {
	Identity         driver.Identity
	IdentityFromMeta driver.Identity
	AuditInfo        []byte
}

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

// TokenDataOpening contains the opening of the TokenData.
// TokenData is a Pedersen commitment to token type and Value.
type TokenDataOpening struct {
	TokenType token2.Type
	Value     *math.Zr
	BF        *math.Zr
}

// OwnerOpening contains the information that allows the auditor to identify the owner.
type OwnerOpening struct {
	OwnerInfo []byte
}

// Auditor inspects zkat tokens and their owners.
type Auditor struct {
	Logger logging.Logger
	Tracer trace.Tracer
	// Owner Identity InfoMatcher
	InfoMatcher InfoMatcher
	// Auditor's signing identity
	Signer SigningIdentity
	// Pedersen generators used to compute TokenData
	PedersenParams []*math.G1
	// Elliptic curve
	Curve *math.Curve
}

func NewAuditor(logger logging.Logger, tracer trace.Tracer, infoMatcher InfoMatcher, pp []*math.G1, signer SigningIdentity, c *math.Curve) *Auditor {
	return &Auditor{
		Logger:         logger,
		Tracer:         tracer,
		InfoMatcher:    infoMatcher,
		PedersenParams: pp,
		Signer:         signer,
		Curve:          c,
	}
}

// Endorse is called to sign a valid token request
func (a *Auditor) Endorse(tokenRequest *driver.TokenRequest, txID string) ([]byte, error) {
	if tokenRequest == nil {
		return nil, errors.Errorf("audit of tx [%s] failed: : token request is nil", txID)
	}
	// Marshal tokenRequest
	bytes, err := tokenRequest.MarshalToMessageToSign([]byte(txID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling token request [%s]", txID)
	}
	// Sign
	a.Logger.Debugf("Endorse [%s][%s]", utils.Hashable(bytes).String(), txID)
	if a.Signer == nil {
		return nil, errors.Errorf("audit of tx [%s] failed: signer is nil", txID)
	}
	return a.Signer.Sign(bytes)
}

// Check validates TokenRequest against TokenRequestMetadata
func (a *Auditor) Check(
	ctx context.Context,
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	inputTokens [][]*token.Token,
	txID driver.TokenRequestAnchor,
) error {
	// TODO: inputTokens should be checked against the actions
	// De-obfuscate issue requests
	a.Logger.DebugfContext(ctx, "Get audit info for %d issues", len(tokenRequest.Issues))
	outputsFromIssue, identitiesFromIssue, err := a.GetAuditInfoForIssues(tokenRequest.Issues, tokenRequestMetadata.Issues)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for issues for [%s]", txID)
	}
	// check validity of issue requests
	a.Logger.DebugfContext(ctx, "Check %d issue outputs", len(outputsFromIssue))
	err = a.CheckIssueRequests(ctx, outputsFromIssue, txID)
	if err != nil {
		return errors.Wrapf(err, "failed checking issues for [%s]", txID)
	}
	for i, id := range identitiesFromIssue {
		err = a.InspectIdentity(ctx, a.InfoMatcher, &id, i)
		if err != nil {
			return errors.Wrapf(err, "failed checking identity for issue [%s]", txID)
		}
	}
	// De-obfuscate transfer requests
	a.Logger.DebugfContext(ctx, "Get audit info for %d transfers", len(tokenRequest.Transfers))
	auditableInputs, outputsFromTransfer, err := a.GetAuditInfoForTransfers(tokenRequest.Transfers, tokenRequestMetadata.Transfers, inputTokens)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for transfers for [%s]", txID)
	}
	// check validity of transfer requests
	a.Logger.DebugfContext(ctx, "Check %d transfer outputs", len(outputsFromTransfer))
	if err := a.CheckTransferRequests(ctx, auditableInputs, outputsFromTransfer, txID); err != nil {
		return errors.Wrapf(err, "failed checking transfers [%s]", txID)
	}

	return nil
}

// CheckTransferRequests verifies that the commitments in transfer inputs and outputs match the information provided in the clear.
func (a *Auditor) CheckTransferRequests(ctx context.Context, inputs [][]*InspectableToken, outputsFromTransfer [][]*InspectableToken, txID driver.TokenRequestAnchor) error {
	for k, transferred := range outputsFromTransfer {
		err := a.InspectOutputs(ctx, transferred)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

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
	// Inspect
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
	if len(a.PedersenParams) != 3 {
		return errors.Errorf("length of Pedersen basis != 3")
	}
	if output == nil || output.Data.Data == nil {
		return errors.Errorf("invalid output at index [%d]", index)
	}
	tokenComm := commit([]*math.Zr{
		a.Curve.HashToZr([]byte(output.Data.TokenType)),
		output.Data.Value,
		output.Data.BF,
	}, a.PedersenParams, a.Curve)
	if !tokenComm.Equals(output.Data.Data) {
		return errors.Errorf("output at index [%d] does not match the provided opening", index)
	}
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

		if !input.Identity.Identity.IsNone() {
			if err := a.InspectIdentity(ctx, a.InfoMatcher, &input.Identity, i); err != nil {
				return errors.Wrapf(err, "failed inspecting input at index [%d]", i)
			}
		}
	}
	return nil
}

// InspectIdentity verifies that the audit info matches the token owner
func (a *Auditor) InspectIdentity(ctx context.Context, matcher InfoMatcher, identity *InspectableIdentity, index int) error {
	if identity.Identity.IsNone() {
		return errors.Errorf("identity at index [%d] is nil, cannot inspect it", index)
	}
	if len(identity.AuditInfo) == 0 {
		return errors.Errorf("failed to inspect identity at index [%d]: audit info is nil", index)
	}
	if len(identity.IdentityFromMeta) != 0 {
		// enforce equality
		if !bytes.Equal(identity.IdentityFromMeta, identity.Identity) {
			return errors.Errorf("failed to inspect identity at index [%d]: identity does not match the identity form metadata", index)
		}
	}
	if err := matcher.MatchIdentity(ctx, identity.Identity, identity.AuditInfo); err != nil {
		return errors.Wrapf(err, "owner at index [%d] does not match the provided opening", index)
	}
	return nil
}

// GetAuditInfoForIssues returns an array of InspectableToken for each issue action
// It takes a deserializer, an array of serialized issue actions and an array of issue metadata.
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
			if ia.Outputs[i].IsRedeem() {
				return nil, nil, errors.Errorf("issue cannot redeem tokens")
			}
			if len(o.Receivers) == 0 {
				return nil, nil, errors.Errorf("issue must have at least one receiver")
			}
			// TODO: check that o.Receivers contains not-nil elements
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
// It takes a deserializer, an array of serialized transfer actions and an array of transfer metadata.
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
		auditableInputs[k] = make([]*InspectableToken, len(transferMetadata.Inputs))
		for i := range len(transferMetadata.Inputs) {
			var err error
			if inputs[k][i] == nil {
				return nil, nil, errors.Errorf("input[%d][%d] is nil", k, i)
			}
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
		outputs[k] = make([]*InspectableToken, len(ta.Outputs))
		for i := range len(ta.Outputs) {
			if ta.Outputs[i] == nil {
				return nil, nil, errors.Errorf("output token at index [%d] is nil", i)
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

func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) *math.G1 {
	com := c.NewG1()
	for i := range vector {
		com.Add(generators[i].Mul(vector[i]))
	}
	return com
}
