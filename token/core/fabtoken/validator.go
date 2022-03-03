/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"bytes"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type Validator struct {
	pp           *PublicParams
	deserializer driver.Deserializer
}

func NewValidator(pp *PublicParams, deserializer driver.Deserializer) *Validator {
	return &Validator{
		pp:           pp,
		deserializer: deserializer,
	}
}

func (v *Validator) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, binding string, tr *driver.TokenRequest) ([]interface{}, error) {
	if err := v.VerifyAuditorSignature(signatureProvider); err != nil {
		return nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", binding)
	}
	ia, ta, err := UnmarshalIssueTransferActions(tr, binding)
	if err != nil {
		return nil, err
	}
	err = v.VerifyIssues(ia, signatureProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify issuers' signatures [%s]", binding)
	}
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

	return actions, nil
}

func (v *Validator) VerifyTokenRequestFromRaw(getState driver.GetStateFnc, binding string, raw []byte) ([]interface{}, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty token request")
	}
	tr := &driver.TokenRequest{}
	err := tr.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	// Prepare message expected to be signed
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
	if len(v.pp.AuditorIdentity()) != 0 {
		signatures = append(signatures, tr.AuditorSignatures...)
		signatures = append(signatures, tr.Signatures...)
	} else {
		signatures = tr.Signatures
	}

	backend := &backend{
		getState:   getState,
		message:    signed,
		signatures: signatures,
	}
	return v.VerifyTokenRequest(backend, backend, binding, tr)
}

func (v *Validator) VerifyAuditorSignature(signatureProvider driver.SignatureProvider) error {
	if v.pp.AuditorIdentity() != nil {
		verifier, err := v.deserializer.GetAuditorVerifier(v.pp.AuditorIdentity())
		if err != nil {
			return errors.Errorf("failed to deserialize auditor's public key")
		}

		return signatureProvider.HasBeenSignedBy(v.pp.AuditorIdentity(), verifier)
	}
	return nil
}

func (v *Validator) VerifyIssues(issues []*IssueAction, signatureProvider driver.SignatureProvider) error {
	for _, issue := range issues {
		if err := v.verifyIssue(issue); err != nil {
			return errors.Wrapf(err, "failed to verify issue action")
		}

		issuers := v.pp.Issuers
		if len(issuers) != 0 {
			// Check that issue.Issuers is in issuers
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

		verifier, err := v.deserializer.GetIssuerVerifier(issue.Issuer)
		if err != nil {
			return errors.Wrapf(err, "failed getting verifier for [%s]", issue.Issuer.String())
		}
		if err := signatureProvider.HasBeenSignedBy(issue.Issuer, verifier); err != nil {
			return errors.Wrapf(err, "failed verifying signature")
		}
	}
	return nil
}

func (v *Validator) VerifyTransfers(ledger driver.Ledger, transferActions []*TransferAction, signatureProvider driver.SignatureProvider) error {
	logger.Debugf("check sender start...")
	defer logger.Debugf("check sender finished.")
	for i, t := range transferActions {
		inputTokens, err := RetrieveInputsFromTransferAction(t, ledger)
		if err != nil {
			return err
		}
		err = v.CheckSendersSignatures(inputTokens, i, signatureProvider)
		if err != nil {
			return err
		}
		if err := v.VerifyTransfer(inputTokens, t); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action")
		}
	}
	return nil
}

func (v *Validator) verifyIssue(issue driver.IssueAction) error {
	if issue.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	for _, output := range issue.GetOutputs() {
		out := output.(*TransferOutput).Output
		q, err := token2.ToQuantity(out.Quantity, 64)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		zero := token2.NewZeroQuantity(64)
		if q.Cmp(zero) == 0 {
			return errors.Errorf("quantity is zero")
		}
	}
	return nil
}

func (v *Validator) VerifyTransfer(inputTokens []*token2.Token, tr driver.TransferAction) error {
	if tr.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	if len(inputTokens) == 0 {
		return errors.Errorf("there is no input")
	}
	if inputTokens[0] == nil {
		return errors.Errorf("first input is nil")
	}
	typ := inputTokens[0].Type
	inputSum := token2.NewZeroQuantity(64)
	outputSum := token2.NewZeroQuantity(64)
	for i, input := range inputTokens {
		if input == nil {
			return errors.Errorf("input %d is nil", i)
		}
		q, err := token2.ToQuantity(input.Quantity, 64)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", input.Quantity)
		}
		inputSum.Add(q)
		if input.Type != typ {
			return errors.Errorf("input type %s does not match type %s", input.Type, typ)
		}
	}
	for _, output := range tr.GetOutputs() {
		out := output.(*TransferOutput).Output
		q, err := token2.ToQuantity(out.Quantity, 64)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		outputSum.Add(q)
		if out.Type != typ {
			return errors.Errorf("output type %s does not match type %s", out.Type, typ)
		}
	}
	if inputSum.Cmp(outputSum) != 0 {
		return errors.Errorf("input sum %v does not match output sum %v", inputSum, outputSum)
	}
	return nil
}

type backend struct {
	getState   driver.GetStateFnc
	message    []byte
	index      int
	signatures [][]byte
}

func (b *backend) HasBeenSignedBy(id view.Identity, verifier driver.Verifier) error {
	if b.index >= len(b.signatures) {
		return errors.Errorf("invalid state, insufficient number of signatures")
	}
	sigma := b.signatures[b.index]
	b.index++

	return verifier.Verify(b.message, sigma)
}

func (b *backend) GetState(key string) ([]byte, error) {
	return b.getState(key)
}

func (b *backend) Signatures() [][]byte {
	return b.signatures
}

func UnmarshalIssueTransferActions(tr *driver.TokenRequest, binding string) ([]*IssueAction, []*TransferAction, error) {
	ia, err := unmarshalIssueActions(tr.Issues)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to retrieve issue actions [%s]", binding)
	}
	ta, err := unmarshalTransferActions(tr.Transfers)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to retrieve transfer actions [%s]", binding)
	}
	return ia, ta, nil
}

func unmarshalTransferActions(raw [][]byte) ([]*TransferAction, error) {
	res := make([]*TransferAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ta := &TransferAction{}
		if err := ta.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ta
	}
	return res, nil
}

func unmarshalIssueActions(raw [][]byte) ([]*IssueAction, error) {
	res := make([]*IssueAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ia := &IssueAction{}
		if err := ia.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ia
	}
	return res, nil
}

func (v *Validator) CheckSendersSignatures(inputTokens []*token2.Token, actionIndex int, signatureProvider driver.SignatureProvider) error {
	for _, tok := range inputTokens {
		logger.Debugf("check sender [%d][%s]", actionIndex, view.Identity(tok.Owner.Raw).UniqueID())
		verifier, err := v.deserializer.GetOwnerVerifier(tok.Owner.Raw)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing owner [%d][%v][%s]", actionIndex, tok, view.Identity(tok.Owner.Raw).UniqueID())
		}
		logger.Debugf("signature verification [%d][%v][%s]", actionIndex, tok, view.Identity(tok.Owner.Raw).UniqueID())
		if err := signatureProvider.HasBeenSignedBy(tok.Owner.Raw, verifier); err != nil {
			return errors.Wrapf(err, "failed signature verification [%d][%v][%s]", actionIndex, tok, view.Identity(tok.Owner.Raw).UniqueID())
		}
	}
	return nil
}

func RetrieveInputsFromTransferAction(t *TransferAction, ledger driver.Ledger) ([]*token2.Token, error) {
	var inputTokens []*token2.Token
	inputs, err := t.GetInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve input IDs")
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
