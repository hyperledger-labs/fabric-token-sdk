/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type MetadataCounterID = string

const (
	TokenRequestToSign     driver.ValidationAttributeID = "trs"
	TokenRequestSignatures driver.ValidationAttributeID = "sigs"
)

type Context[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] struct {
	Logger            logging.Logger
	PP                P
	Anchor            driver.TokenRequestAnchor
	TokenRequest      *driver.TokenRequest
	Deserializer      DS
	SignatureProvider driver.SignatureProvider
	Signatures        [][]byte
	InputTokens       []T
	TransferAction    TA
	IssueAction       IA
	Ledger            driver.Ledger
	MetadataCounter   map[MetadataCounterID]int
	Attributes        driver.ValidationAttributes
}

func (c *Context[P, T, TA, IA, DS]) CountMetadataKey(key string) {
	c.MetadataCounter[key] = c.MetadataCounter[key] + 1
}

type ValidateTransferFunc[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] func(c context.Context, ctx *Context[P, T, TA, IA, DS]) error

type ValidateIssueFunc[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] func(c context.Context, ctx *Context[P, T, TA, IA, DS]) error

type ValidateAuditingFunc[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] func(c context.Context, ctx *Context[P, T, TA, IA, DS]) error

type ActionDeserializer[TA driver.TransferAction, IA driver.IssueAction] interface {
	DeserializeActions(tr *driver.TokenRequest) ([]IA, []TA, error)
}

type Validator[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] struct {
	Logger             logging.Logger
	PublicParams       P
	Deserializer       DS
	ActionDeserializer ActionDeserializer[TA, IA]

	AuditingValidators []ValidateAuditingFunc[P, T, TA, IA, DS]
	TransferValidators []ValidateTransferFunc[P, T, TA, IA, DS]
	IssueValidators    []ValidateIssueFunc[P, T, TA, IA, DS]
}

func NewValidator[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](
	Logger logging.Logger,
	publicParams P,
	deserializer DS,
	actionDeserializer ActionDeserializer[TA, IA],
	transferValidators []ValidateTransferFunc[P, T, TA, IA, DS],
	issueValidators []ValidateIssueFunc[P, T, TA, IA, DS],
	auditingValidators []ValidateAuditingFunc[P, T, TA, IA, DS],
) *Validator[P, T, TA, IA, DS] {
	return &Validator[P, T, TA, IA, DS]{
		Logger:             Logger,
		PublicParams:       publicParams,
		Deserializer:       deserializer,
		ActionDeserializer: actionDeserializer,
		TransferValidators: transferValidators,
		IssueValidators:    issueValidators,
		AuditingValidators: auditingValidators,
	}
}

func (v *Validator[P, T, TA, IA, DS]) VerifyTokenRequestFromRaw(ctx context.Context, getState driver.GetStateFnc, anchor driver.TokenRequestAnchor, raw []byte) ([]interface{}, driver.ValidationAttributes, error) {
	logger.DebugfContext(ctx, "Verify token request from raw")
	if len(raw) == 0 {
		return nil, nil, errors.New("empty token request")
	}
	tr := &driver.TokenRequest{}
	err := tr.FromBytes(raw)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	// Prepare message expected to be signed
	signed, err := tr.MarshalToMessageToSign([]byte(anchor))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal signed token request")
	}
	signatures := make([][]byte, 0, len(tr.AuditorSignatures)+len(tr.Signatures))
	for _, sig := range tr.AuditorSignatures {
		signatures = append(signatures, sig.Signature)
	}
	signatures = append(signatures, tr.Signatures...)

	attributes := make(driver.ValidationAttributes)
	attributes[TokenRequestToSign] = signed
	attributes[TokenRequestSignatures], err = json.Marshal(signatures)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal token request signatures")
	}

	backend := NewBackend(v.Logger, getState, signed, signatures)

	return v.VerifyTokenRequest(ctx, backend, backend, anchor, tr, attributes)
}

func (v *Validator[P, T, TA, IA, DS]) VerifyTokenRequest(
	ctx context.Context,
	ledger driver.Ledger,
	signatureProvider driver.SignatureProvider,
	anchor driver.TokenRequestAnchor,
	tr *driver.TokenRequest,
	attributes driver.ValidationAttributes,
) ([]interface{}, driver.ValidationAttributes, error) {
	if err := v.VerifyAuditing(ctx, anchor, tr, ledger, signatureProvider, attributes); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", anchor)
	}
	ia, ta, err := v.ActionDeserializer.DeserializeActions(tr)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to unmarshal actions [%s]", anchor)
	}
	err = v.verifyIssues(ctx, anchor, tr, ledger, ia, signatureProvider, attributes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verify issue actions [%s]", anchor)
	}
	err = v.verifyTransfers(ctx, anchor, tr, ledger, ta, signatureProvider, attributes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verify transfer actions [%s]", anchor)
	}

	var actions []interface{}
	for _, action := range ia {
		actions = append(actions, action)
	}
	for _, action := range ta {
		actions = append(actions, action)
	}

	return actions, attributes, nil
}

func (v *Validator[P, T, TA, IA, DS]) UnmarshalActions(raw []byte) ([]interface{}, error) {
	tr := &driver.TokenRequest{}
	err := tr.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	ia, ta, err := v.ActionDeserializer.DeserializeActions(tr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal actions")
	}
	var res []interface{}
	for _, action := range ia {
		res = append(res, action)
	}
	for _, action := range ta {
		res = append(res, action)
	}

	return res, nil
}

func (v *Validator[P, T, TA, IA, DS]) verifyIssues(
	ctx context.Context,
	anchor driver.TokenRequestAnchor,
	tokenRequest *driver.TokenRequest,
	ledger driver.Ledger,
	issues []IA,
	signatureProvider driver.SignatureProvider,
	attributes driver.ValidationAttributes,
) error {
	for i, issue := range issues {
		if err := v.VerifyIssue(ctx, anchor, tokenRequest, issue, ledger, signatureProvider, attributes); err != nil {
			return errors.Wrapf(err, "failed to verify issue action at [%d]", i)
		}
	}

	return nil
}

func (v *Validator[P, T, TA, IA, DS]) VerifyIssue(
	ctx context.Context,
	anchor driver.TokenRequestAnchor,
	tokenRequest *driver.TokenRequest,
	action IA,
	ledger driver.Ledger,
	signatureProvider driver.SignatureProvider,
	attributes driver.ValidationAttributes,
) error {
	context := &Context[P, T, TA, IA, DS]{
		Logger:            v.Logger,
		PP:                v.PublicParams,
		Anchor:            anchor,
		TokenRequest:      tokenRequest,
		Deserializer:      v.Deserializer,
		IssueAction:       action,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		MetadataCounter:   map[string]int{},
		Attributes:        attributes,
	}
	for _, v := range v.IssueValidators {
		if err := v(ctx, context); err != nil {
			return err
		}
	}

	// Check that all metadata have been validated
	counter := 0
	for k, c := range context.MetadataCounter {
		if c > 1 {
			return errors.Errorf("metadata key [%s] appeared more than one time", k)
		}
		counter += c
	}
	if len(action.GetMetadata()) != counter {
		return errors.Errorf("more metadata than those validated [%d]!=[%d], [%v]!=[%v]", len(action.GetMetadata()), counter, action.GetMetadata(), context.MetadataCounter)
	}

	return nil
}

func (v *Validator[P, T, TA, IA, DS]) verifyTransfers(
	ctx context.Context,
	anchor driver.TokenRequestAnchor,
	tokenRequest *driver.TokenRequest,
	ledger driver.Ledger,
	transferActions []TA,
	signatureProvider driver.SignatureProvider,
	attributes driver.ValidationAttributes,
) error {
	v.Logger.Debugf("check sender start...")
	defer v.Logger.Debugf("check sender finished.")
	for i, action := range transferActions {
		if err := v.VerifyTransfer(ctx, anchor, tokenRequest, action, ledger, signatureProvider, attributes); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action at [%d]", i)
		}
	}

	return nil
}

func (v *Validator[P, T, TA, IA, DS]) VerifyTransfer(
	ctx context.Context,
	anchor driver.TokenRequestAnchor,
	tokenRequest *driver.TokenRequest,
	action TA,
	ledger driver.Ledger,
	signatureProvider driver.SignatureProvider,
	attributes driver.ValidationAttributes,
) error {
	context := &Context[P, T, TA, IA, DS]{
		Logger:            v.Logger,
		PP:                v.PublicParams,
		Anchor:            anchor,
		TokenRequest:      tokenRequest,
		Deserializer:      v.Deserializer,
		TransferAction:    action,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		MetadataCounter:   map[MetadataCounterID]int{},
		Attributes:        attributes,
	}
	for _, v := range v.TransferValidators {
		if err := v(ctx, context); err != nil {
			return err
		}
	}

	// Check that all metadata have been validated
	counter := 0
	for k, c := range context.MetadataCounter {
		if c > 1 {
			return errors.Errorf("metadata key [%s] appeared more than one time", k)
		}
		counter += c
	}
	if len(action.GetMetadata()) != counter {
		return errors.Errorf("more metadata than those validated [%d]!=[%d], [%v]!=[%v]", len(action.GetMetadata()), counter, action.GetMetadata(), context.MetadataCounter)
	}

	return nil
}

func (v *Validator[P, T, TA, IA, DS]) VerifyAuditing(
	ctx context.Context,
	anchor driver.TokenRequestAnchor,
	tokenRequest *driver.TokenRequest,
	ledger driver.Ledger,
	signatureProvider driver.SignatureProvider,
	attributes driver.ValidationAttributes,
) error {
	context := &Context[P, T, TA, IA, DS]{
		Logger:            v.Logger,
		PP:                v.PublicParams,
		Anchor:            anchor,
		TokenRequest:      tokenRequest,
		Deserializer:      v.Deserializer,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		Attributes:        attributes,
	}
	for _, v := range v.AuditingValidators {
		if err := v(ctx, context); err != nil {
			return err
		}
	}

	return nil
}

func IsAnyNil[T any](args ...*T) bool {
	for _, arg := range args {
		if arg == nil {
			return true
		}
	}

	return false
}
