/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Validator struct {
	pp *PublicParams
}

func NewValidator(pp *PublicParams) *Validator {
	return &Validator{pp: pp}
}

func (v *Validator) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, binding string, tr *driver.TokenRequest) ([]interface{}, error) {
	if err := v.verifyAuditorSignature(signatureProvider); err != nil {
		return nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", binding)
	}
	ia, ta, err := UnmarshalIssueTransferActions(tr, binding)
	if err != nil {
		return nil, err
	}
	err = v.verifyIssues(ia, signatureProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify issuers' signatures [%s]", binding)
	}
	err = v.verifyTransfers(ledger, ta, signatureProvider)
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

func (v *Validator) VerifyTokenRequestFromRaw(getState driver.GetStateFnc, binding string, raw []byte) ([]interface{}, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty token request")
	}
	tr := &driver.TokenRequest{}
	err := json.Unmarshal(raw, tr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	// Prepare message expected to be signed
	// TODO: encapsulate this somewhere
	req := &driver.TokenRequest{}
	req.Transfers = tr.Transfers
	req.Issues = tr.Issues
	bytes, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal signed token request"+err.Error())
	}

	logger.Debugf("cc tx-id [%s][%s]", hash.Hashable(bytes).String(), binding)
	signed := append(bytes, []byte(binding)...)
	var signatures [][]byte
	if len(v.pp.Auditor) != 0 {
		signatures = append(signatures, tr.AuditorSignature)
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

func (v *Validator) verifyAuditorSignature(signatureProvider driver.SignatureProvider) error {
	if v.pp.Auditor != nil {
		identityDeserializer := &fabric.MSPX509IdentityDeserializer{}
		verifier, err := identityDeserializer.GetVerifier(v.pp.Auditor)
		if err != nil {
			return errors.Errorf("failed to deserialize auditor's public key")
		}

		return signatureProvider.HasBeenSignedBy(v.pp.Auditor, verifier)
	}
	return nil
}

func (v *Validator) verifyIssues(issues []*IssueAction, signatureProvider driver.SignatureProvider) error {
	for _, issue := range issues {
		if err := v.verifyIssue(issue); err != nil {
			return errors.Wrapf(err, "failed to verify issue action")
		}

		identityDeserializer := NewRawOwnerIdentityDeserializer()
		verifier, err := identityDeserializer.GetVerifier(issue.Issuer)
		if err != nil {
			return errors.Wrapf(err, "failed getting verifier for [%s]", issue.Issuer.String())
		}
		if err := signatureProvider.HasBeenSignedBy(issue.Issuer, verifier); err != nil {
			return errors.Wrapf(err, "failed verifying signature")
		}
	}
	return nil
}

func (v *Validator) verifyTransfers(ledger driver.Ledger, transferActions []*TransferAction, signatureProvider driver.SignatureProvider) error {
	identityDeserializer := NewRawOwnerIdentityDeserializer()
	logger.Debugf("check sender start...")
	defer logger.Debugf("check sender finished.")
	for i, t := range transferActions {
		var inputTokens [][]byte
		inputs, err := t.GetInputs()
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve input IDs")
		}
		for _, in := range inputs {
			logger.Debugf("load token [%d][%s]", i, in)
			bytes, err := ledger.GetState(in)
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve input to spend [%s]", in)
			}
			if len(bytes) == 0 {
				return errors.Errorf("finput to spend [%s] does not exists", in)
			}
			inputTokens = append(inputTokens, bytes)
			tok := &token2.Token{}
			err = json.Unmarshal(bytes, tok)
			if err != nil {
				return errors.Wrapf(err, "failed to deserialize input to spend [%s]", in)
			}
			logger.Debugf("check sender [%d][%s]", i, view.Identity(tok.Owner.Raw).UniqueID())

			verifier, err := identityDeserializer.GetVerifier(tok.Owner.Raw)
			if err != nil {
				return errors.Wrapf(err, "failed deserializing owner [%d][%s][%s]", i, in, view.Identity(tok.Owner.Raw).UniqueID())
			}
			logger.Debugf("signature verification [%d][%s][%s]", i, in, view.Identity(tok.Owner.Raw).UniqueID())
			if err := signatureProvider.HasBeenSignedBy(tok.Owner.Raw, verifier); err != nil {
				return errors.Wrapf(err, "failed signature verification [%d][%s][%s]", i, in, view.Identity(tok.Owner.Raw).UniqueID())
			}
		}
		if err := v.verifyTransfer(inputTokens, t); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action")
		}
	}
	return nil
}

func (v *Validator) verifyIssue(issue driver.IssueAction) error {
	return nil
}

func (v *Validator) verifyTransfer(inputTokens [][]byte, tr driver.TransferAction) error {
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
