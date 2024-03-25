/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

const (
	TokenRequestToSign = "trs"
)

type Context[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction] struct {
	PP                P
	Deserializer      driver.Deserializer
	SignatureProvider driver.SignatureProvider
	Signatures        [][]byte
	InputTokens       []T
	TransferAction    TA
	IssueAction       IA
	Ledger            driver.Ledger
	MetadataCounter   map[string]int
	Attributes        map[string][]byte
}

func (c *Context[P, T, TA, IA]) CountMetadataKey(key string) {
	c.MetadataCounter[key] = c.MetadataCounter[key] + 1
}

type ValidateTransferFunc[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction] func(ctx *Context[P, T, TA, IA]) error

type ValidateIssueFunc[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction] func(ctx *Context[P, T, TA, IA]) error

type ActionDeserializer[TA driver.TransferAction, IA driver.IssueAction] interface {
	DeserializeActions(tr *driver.TokenRequest) ([]IA, []TA, error)
}

type Validator[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction] struct {
	Logger             *flogging.FabricLogger
	PublicParams       P
	Deserializer       driver.Deserializer
	ActionDeserializer ActionDeserializer[TA, IA]
	TransferValidators []ValidateTransferFunc[P, T, TA, IA]
	IssueValidators    []ValidateIssueFunc[P, T, TA, IA]
	Serializer         driver.Serializer
}

func NewValidator[P driver.PublicParameters, T any, TA driver.TransferAction, IA driver.IssueAction](
	Logger *flogging.FabricLogger,
	publicParams P,
	deserializer driver.Deserializer,
	actionDeserializer ActionDeserializer[TA, IA],
	transferValidators []ValidateTransferFunc[P, T, TA, IA],
	issueValidators []ValidateIssueFunc[P, T, TA, IA],
	serializer driver.Serializer,
) *Validator[P, T, TA, IA] {
	return &Validator[P, T, TA, IA]{
		Logger:             Logger,
		PublicParams:       publicParams,
		Deserializer:       deserializer,
		ActionDeserializer: actionDeserializer,
		TransferValidators: transferValidators,
		IssueValidators:    issueValidators,
		Serializer:         serializer,
	}
}

func (v *Validator[P, T, TA, IA]) VerifyTokenRequestFromRaw(getState driver.GetStateFnc, anchor string, raw []byte) ([]interface{}, map[string][]byte, error) {
	if len(raw) == 0 {
		return nil, nil, errors.New("empty token request")
	}
	tr := &driver.TokenRequest{}
	err := tr.FromBytes(raw)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	// Prepare message expected to be signed
	req := &driver.TokenRequest{}
	req.Transfers = tr.Transfers
	req.Issues = tr.Issues
	raqRaw, err := req.Bytes()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal signed token request")
	}

	v.Logger.Debugf("cc tx-id [%s][%s]", hash.Hashable(raqRaw).String(), anchor)
	signed := append(raqRaw, []byte(anchor)...)
	var signatures [][]byte
	if len(v.PublicParams.Auditors()) != 0 {
		signatures = append(signatures, tr.AuditorSignatures...)
		signatures = append(signatures, tr.Signatures...)
	} else {
		signatures = tr.Signatures
	}

	attributes := make(map[string][]byte)
	attributes[TokenRequestToSign], err = v.Serializer.MarshalTokenRequestToSign(req, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal signed token request")
	}

	backend := NewBackend(getState, signed, signatures)
	return v.VerifyTokenRequest(backend, backend, anchor, tr, attributes)
}

func (v *Validator[P, T, TA, IA]) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, anchor string, tr *driver.TokenRequest, attributes map[string][]byte) ([]interface{}, map[string][]byte, error) {
	if err := v.verifyAuditorSignature(signatureProvider, attributes); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", anchor)
	}
	ia, ta, err := v.ActionDeserializer.DeserializeActions(tr)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to unmarshal actions [%s]", anchor)
	}
	err = v.verifyIssues(ledger, ia, signatureProvider, attributes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verify issuers' signatures [%s]", anchor)
	}
	err = v.verifyTransfers(ledger, ta, signatureProvider, attributes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to verify senders' signatures [%s]", anchor)
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

func (v *Validator[P, T, TA, IA]) UnmarshalActions(raw []byte) ([]interface{}, error) {
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

func (v *Validator[P, T, TA, IA]) verifyAuditorSignature(signatureProvider driver.SignatureProvider, attributes map[string][]byte) error {
	if len(v.PublicParams.Auditors()) != 0 {
		auditor := v.PublicParams.Auditors()[0]
		verifier, err := v.Deserializer.GetAuditorVerifier(auditor)
		if err != nil {
			return errors.Errorf("failed to deserialize auditor's public key")
		}
		_, err = signatureProvider.HasBeenSignedBy(auditor, verifier)
		return err
	}
	return nil
}

func (v *Validator[P, T, TA, IA]) verifyIssues(ledger driver.Ledger, issues []IA, signatureProvider driver.SignatureProvider, attributes map[string][]byte) error {
	for _, issue := range issues {
		if err := v.verifyIssue(issue, ledger, signatureProvider, attributes); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action")
		}
	}
	return nil
}

func (v *Validator[P, T, TA, IA]) verifyIssue(tr IA, ledger driver.Ledger, signatureProvider driver.SignatureProvider, attributes map[string][]byte) error {
	context := &Context[P, T, TA, IA]{
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

func (v *Validator[P, T, TA, IA]) verifyTransfers(ledger driver.Ledger, transferActions []TA, signatureProvider driver.SignatureProvider, attributes map[string][]byte) error {
	v.Logger.Debugf("check sender start...")
	defer v.Logger.Debugf("check sender finished.")
	for _, action := range transferActions {
		if err := v.verifyTransfer(action, ledger, signatureProvider, attributes); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action")
		}
	}
	return nil
}

func (v *Validator[P, T, TA, IA]) verifyTransfer(tr TA, ledger driver.Ledger, signatureProvider driver.SignatureProvider, attributes map[string][]byte) error {
	context := &Context[P, T, TA, IA]{
		PP:                v.PublicParams,
		Deserializer:      v.Deserializer,
		TransferAction:    tr,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		MetadataCounter:   map[string]int{},
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
