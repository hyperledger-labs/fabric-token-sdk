/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"bytes"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// Validator checks the validity of fabtoken TokenRequest
type Validator struct {
	// fabtoken public parameters
	pp *PublicParams
	// deserializer for identities
	deserializer driver.Deserializer
	// transferValidators for performing transfer action validation
	transferValidators []ValidateTransferFunc
}

// NewValidator initializes a Validator with the passed parameters
func NewValidator(pp *PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) (*Validator, error) {
	if pp == nil {
		return nil, errors.New("please provide a non-nil public parameters")
	}
	if deserializer == nil {
		return nil, errors.New("please provide a non-nil deserializer")
	}

	transferValidators := []ValidateTransferFunc{
		TransferSignatureValidate,
		TransferBalanceValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)
	v := &Validator{
		pp:                 pp,
		deserializer:       deserializer,
		transferValidators: transferValidators,
	}
	return v, nil
}

// VerifyTokenRequest validates the passed token request against data in the ledger, the signature provided and the binding
func (v *Validator) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, binding string, tr *driver.TokenRequest) ([]interface{}, error) {
	// validate arguments
	if ledger == nil {
		return nil, errors.New("please provide a non-nil ledger")
	}
	if signatureProvider == nil {
		return nil, errors.New("please provide a non-nil signature provider")
	}
	if len(binding) == 0 {
		return nil, errors.New("please provide a non-empty binding")
	}
	if tr == nil {
		return nil, errors.New("please provide a non-nil token request")
	}

	// check if the token request is signed by the authorized auditor
	if err := v.VerifyAuditorSignature(signatureProvider); err != nil {
		return nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", binding)
	}
	// get issue and transfer actions from the token request
	ia, ta, err := UnmarshalIssueTransferActions(tr, binding)
	if err != nil {
		return nil, err
	}
	// verify issue actions
	err = v.VerifyIssues(ia, signatureProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify issuers' signatures [%s]", binding)
	}
	// verify transfer actions
	err = v.VerifyTransfers(ledger, ta, signatureProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify senders' signatures [%s]", binding)
	}

	var actions []interface{}
	for _, action := range ia {
		actions = append(actions, action)
	}
	for _, action := range ta {
		actions = append(actions, action)
	}

	// actions are returned and will be used later to update the ledger
	return actions, nil
}

// VerifyTokenRequestFromRaw validates the raw token request
func (v *Validator) VerifyTokenRequestFromRaw(getState driver.GetStateFnc, binding string, raw []byte) ([]interface{}, error) {
	if getState == nil {
		return nil, errors.New("please provide a non-nil get state function")
	}
	if len(binding) == 0 {
		return nil, errors.New("please provide a non-empty binding")
	}
	if len(raw) == 0 {
		return nil, errors.New("empty token request")
	}
	// un-marshal token request
	tr := &driver.TokenRequest{}
	err := tr.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	// Prepare Message expected to be signed
	// TODO: encapsulate this somewhere
	req := &driver.TokenRequest{}
	req.Transfers = tr.Transfers
	req.Issues = tr.Issues
	bytes, err := req.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal signed token request"+err.Error())
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("cc tx-id [%s][%s]", hash.Hashable(bytes).String(), binding)
	}
	signed := append(bytes, []byte(binding)...)
	var signatures [][]byte
	// audit is enabled
	if len(v.pp.AuditorIdentity()) != 0 {
		signatures = append(signatures, tr.AuditorSignatures...)
		signatures = append(signatures, tr.Signatures...)
	} else {
		signatures = tr.Signatures
	}

	backend := common.NewBackend(getState, signed, signatures)
	return v.VerifyTokenRequest(backend, backend, binding, tr)
}

// VerifyAuditorSignature checks if the content of the token request concatenated with the binding
// was signed by the authorized auditor
func (v *Validator) VerifyAuditorSignature(signatureProvider driver.SignatureProvider) error {
	if v.pp.AuditorIdentity() != nil {
		verifier, err := v.deserializer.GetAuditorVerifier(v.pp.AuditorIdentity())
		if err != nil {
			return errors.New("failed to deserialize auditor's public key")
		}

		_, err = signatureProvider.HasBeenSignedBy(v.pp.AuditorIdentity(), verifier)
		return err
	}
	return nil
}

// VerifyIssues checks if the issued tokens are valid and if the content of the token request concatenated
// with the binding was signed by one of the authorized issuers
func (v *Validator) VerifyIssues(issues []*IssueAction, signatureProvider driver.SignatureProvider) error {
	for _, issue := range issues {
		// verify that issue is valid
		if err := v.VerifyIssue(issue); err != nil {
			return errors.Wrap(err, "failed to verify issue action")
		}

		issuers := v.pp.Issuers
		if len(issuers) != 0 {
			// check that issuer of this issue action is authorized
			found := false
			for _, issuer := range issuers {
				if bytes.Equal(issue.Issuer, issuer) {
					found = true
					break
				}
			}
			if !found {
				return errors.Errorf("issuer [%s] is not in issuers", issue.Issuer.String())
			}
		}

		// deserialize verifier for the issuer
		verifier, err := v.deserializer.GetIssuerVerifier(issue.Issuer)
		if err != nil {
			return errors.Wrapf(err, "failed getting verifier for [%s]", issue.Issuer.String())
		}
		// verify if the token request concatenated with the binding was signed by the issuer
		if _, err := signatureProvider.HasBeenSignedBy(issue.Issuer, verifier); err != nil {
			return errors.Wrapf(err, "failed verifying signature")
		}
	}
	return nil
}

// VerifyIssue checks if all outputs in IssueAction are valid (no zero-value outputs)
func (v *Validator) VerifyIssue(issue driver.IssueAction) error {
	if issue.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	for _, output := range issue.GetOutputs() {
		out := output.(*Output).Output
		q, err := token2.ToQuantity(out.Quantity, v.pp.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		zero := token2.NewZeroQuantity(v.pp.QuantityPrecision)
		if q.Cmp(zero) == 0 {
			return errors.Errorf("quantity is zero")
		}
	}
	return nil
}

// VerifyTransfers checks if the created output tokens are valid and if the content of the token request concatenated
// with the binding was signed by the owners of the input tokens
func (v *Validator) VerifyTransfers(ledger driver.Ledger, transferActions []*TransferAction, signatureProvider driver.SignatureProvider) error {
	logger.Debugf("check sender start...")
	defer logger.Debugf("check sender finished.")
	for i, t := range transferActions {
		// get inputs used in the current transfer action
		inputTokens, err := RetrieveInputsFromTransferAction(t, ledger)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve input from transfer action at index %d", i)
		}
		// verify if input tokens and output tokens in the current transfer action have the same type
		// verify if sum of input tokens in the current transfer action equals the sum of output tokens
		// in the current transfer action
		if err := v.VerifyTransfer(ledger, inputTokens, t, signatureProvider); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action at index %d", i)
		}
	}
	return nil
}

// VerifyTransfer checks that sum of inputTokens in TransferAction equals sum of outputs in TransferAction
// It also checks that all outputs and inputs have the same type
func (v *Validator) VerifyTransfer(ledger driver.Ledger, inputTokens []*token2.Token, tr driver.TransferAction, signatureProvider driver.SignatureProvider) error {
	ctx := &Context{
		PP:                v.pp,
		Deserializer:      v.deserializer,
		SignatureProvider: signatureProvider,
		InputTokens:       inputTokens,
		Action:            tr.(*TransferAction),
		Ledger:            ledger,
		MetadataCounter:   map[string]int{},
	}
	for _, validator := range v.transferValidators {
		if err := validator(ctx); err != nil {
			return err
		}
	}

	// Check that all metadata have been validated
	counter := 0
	for k, c := range ctx.MetadataCounter {
		if c > 1 {
			return errors.Errorf("metadata key [%s] appeared more than once", k)
		}
		counter += c
	}
	if len(tr.GetMetadata()) != counter {
		return errors.Errorf("more metadata than those validated [%d]!=[%d]", len(tr.GetMetadata()), counter)
	}
	return nil
}

// RetrieveInputsFromTransferAction retrieves from the passed ledger the inputs identified in TransferAction
func RetrieveInputsFromTransferAction(t *TransferAction, ledger driver.Ledger) ([]*token2.Token, error) {
	var inputTokens []*token2.Token
	inputs, err := t.GetInputs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve input IDs")
	}
	for _, in := range inputs {
		bytes, err := ledger.GetState(in)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve input to spend [%s]", in)
		}
		if len(bytes) == 0 {
			return nil, errors.Errorf("input to spend [%s] does not exists", in)
		}
		tok := &token2.Token{}
		err = json.Unmarshal(bytes, tok)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to deserialize input to spend [%s]", in)
		}
		inputTokens = append(inputTokens, tok)
	}
	return inputTokens, nil
}
