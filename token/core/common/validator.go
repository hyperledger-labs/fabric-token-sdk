/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

type MetadataCounterID = string

const (
	TokenRequestToSign     driver.ValidationAttributeID = "trs"
	TokenRequestSignatures driver.ValidationAttributeID = "sigs"
)

type Context[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] struct {
	Logger            logging.Logger
	PP                P
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

type ValidateTransferFunc[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] func(ctx *Context[P, T, TA, IA, DS]) error

type ValidateIssueFunc[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] func(ctx *Context[P, T, TA, IA, DS]) error

type ActionDeserializer[TA driver.TransferAction, IA driver.IssueAction] interface {
	DeserializeActions(tr *driver.TokenRequest) ([]IA, []TA, error)
}

type Validator[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer] struct {
	Logger             logging.Logger
	PublicParams       P
	Deserializer       DS
	ActionDeserializer ActionDeserializer[TA, IA]
	TransferValidators []ValidateTransferFunc[P, T, TA, IA, DS]
	IssueValidators    []ValidateIssueFunc[P, T, TA, IA, DS]
	Serializer         driver.Serializer
}

func NewValidator[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](
	Logger logging.Logger,
	publicParams P,
	deserializer DS,
	actionDeserializer ActionDeserializer[TA, IA],
	transferValidators []ValidateTransferFunc[P, T, TA, IA, DS],
	issueValidators []ValidateIssueFunc[P, T, TA, IA, DS],
	serializer driver.Serializer,
) *Validator[P, T, TA, IA, DS] {
	return &Validator[P, T, TA, IA, DS]{
		Logger:             Logger,
		PublicParams:       publicParams,
		Deserializer:       deserializer,
		ActionDeserializer: actionDeserializer,
		TransferValidators: transferValidators,
		IssueValidators:    issueValidators,
		Serializer:         serializer,
	}
}

func (v *Validator[P, T, TA, IA, DS]) VerifyTokenRequestFromRaw(ctx context.Context, getState driver.GetStateFnc, anchor string, raw []byte) ([]interface{}, driver.ValidationAttributes, error) {
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
	var signatures [][]byte
	if len(v.PublicParams.Auditors()) != 0 {
		signatures = append(signatures, tr.AuditorSignatures...)
		signatures = append(signatures, tr.Signatures...)
	} else {
		signatures = tr.Signatures
	}

	attributes := make(driver.ValidationAttributes)
	attributes[TokenRequestToSign] = signed
	attributes[TokenRequestSignatures], err = json.Marshal(signatures)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal token request signatures")
	}

	backend := NewBackend(v.Logger, getState, signed, signatures)
	return v.VerifyTokenRequest(backend, backend, anchor, tr, attributes)
}

func (v *Validator[P, T, TA, IA, DS]) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, anchor string, tr *driver.TokenRequest, attributes driver.ValidationAttributes) ([]interface{}, driver.ValidationAttributes, error) {
	if err := v.verifyAuditorSignature(signatureProvider, attributes); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", anchor)
	}
	ia, ta, err := v.ActionDeserializer.DeserializeActions(tr)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to unmarshal actions [%s]", anchor)
	}
	err = v.verifyIssues(ledger, ia, signatureProvider, attributes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verify issue actions [%s]", anchor)
	}
	err = v.verifyTransfers(ledger, ta, signatureProvider, attributes)
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

func (v *Validator[P, T, TA, IA, DS]) verifyAuditorSignature(signatureProvider driver.SignatureProvider, attributes driver.ValidationAttributes) error {
	if len(v.PublicParams.Auditors()) != 0 {
		auditor := v.PublicParams.Auditors()[0]
		verifier, err := v.Deserializer.GetAuditorVerifier(auditor)
		if err != nil {
			return errors.Errorf("failed to deserialize auditor's public key")
		}
		v.Logger.Infof("verify auditor signature for [%s]", auditor)
		_, err = signatureProvider.HasBeenSignedBy(auditor, verifier)
		return err
	}
	return nil
}

func (v *Validator[P, T, TA, IA, DS]) verifyIssues(ledger driver.Ledger, issues []IA, signatureProvider driver.SignatureProvider, attributes driver.ValidationAttributes) error {
	for i, issue := range issues {
		if err := v.verifyIssue(issue, ledger, signatureProvider, attributes); err != nil {
			return errors.Wrapf(err, "failed to verify issue action at [%d]", i)
		}
	}
	return nil
}

func (v *Validator[P, T, TA, IA, DS]) verifyIssue(tr IA, ledger driver.Ledger, signatureProvider driver.SignatureProvider, attributes driver.ValidationAttributes) error {
	context := &Context[P, T, TA, IA, DS]{
		Logger:            v.Logger,
		PP:                v.PublicParams,
		Deserializer:      v.Deserializer,
		IssueAction:       tr,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		MetadataCounter:   map[string]int{},
		Attributes:        attributes,
	}
	for _, v := range v.IssueValidators {
		if err := v(context); err != nil {
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
	if len(tr.GetMetadata()) != counter {
		return errors.Errorf("more metadata than those validated [%d]!=[%d], [%v]!=[%v]", len(tr.GetMetadata()), counter, tr.GetMetadata(), context.MetadataCounter)
	}

	return nil
}

func (v *Validator[P, T, TA, IA, DS]) verifyTransfers(ledger driver.Ledger, transferActions []TA, signatureProvider driver.SignatureProvider, attributes driver.ValidationAttributes) error {
	v.Logger.Debugf("check sender start...")
	defer v.Logger.Debugf("check sender finished.")
	for i, action := range transferActions {
		if err := v.verifyTransfer(action, ledger, signatureProvider, attributes); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action at [%d]", i)
		}
	}
	return nil
}

func (v *Validator[P, T, TA, IA, DS]) verifyTransfer(tr TA, ledger driver.Ledger, signatureProvider driver.SignatureProvider, attributes driver.ValidationAttributes) error {
	context := &Context[P, T, TA, IA, DS]{
		Logger:            v.Logger,
		PP:                v.PublicParams,
		Deserializer:      v.Deserializer,
		TransferAction:    tr,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		MetadataCounter:   map[MetadataCounterID]int{},
		Attributes:        attributes,
	}
	for _, v := range v.TransferValidators {
		if err := v(context); err != nil {
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
	if len(tr.GetMetadata()) != counter {
		return errors.Errorf("more metadata than those validated [%d]!=[%d], [%v]!=[%v]", len(tr.GetMetadata()), counter, tr.GetMetadata(), context.MetadataCounter)
	}

	return nil
}
