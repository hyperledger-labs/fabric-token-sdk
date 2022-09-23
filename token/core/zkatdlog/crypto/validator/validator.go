/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog")

type Validator struct {
	pp                 *crypto.PublicParams
	deserializer       driver.Deserializer
	transferValidators []ValidateTransferFunc
}

func New(pp *crypto.PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferSignatureValidate,
		TransferZKProofValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)
	return &Validator{
		pp:                 pp,
		deserializer:       deserializer,
		transferValidators: transferValidators,
	}
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
		return nil, errors.Wrap(err, "failed to marshal signed token request")
	}

	logger.Debugf("cc tx-id [%s][%s]", hash.Hashable(bytes).String(), binding)
	signed := append(bytes, []byte(binding)...)
	var signatures [][]byte
	if len(v.pp.Auditor) != 0 {
		signatures = append(signatures, tr.AuditorSignatures...)
		signatures = append(signatures, tr.Signatures...)
	} else {
		signatures = tr.Signatures
	}

	backend := common.NewBackend(getState, signed, signatures)
	return v.VerifyTokenRequest(backend, backend, binding, tr)
}

func (v *Validator) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, binding string, tr *driver.TokenRequest) ([]interface{}, error) {
	if err := v.verifyAuditorSignature(signatureProvider); err != nil {
		return nil, errors.Wrapf(err, "failed to verifier auditor's signature [%s]", binding)
	}
	ia, err := v.unmarshalIssueActions(tr.Issues)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve issue actions [%s]", binding)
	}
	ta, err := v.unmarshalTransferActions(tr.Transfers)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve transfer actions [%s]", binding)
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

func (v *Validator) unmarshalTransferActions(raw [][]byte) ([]driver.TransferAction, error) {
	res := make([]driver.TransferAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ta := &transfer.TransferAction{}
		if err := ta.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ta
	}
	return res, nil
}

func (v *Validator) unmarshalIssueActions(raw [][]byte) ([]driver.IssueAction, error) {
	res := make([]driver.IssueAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ia := &issue2.IssueAction{}
		if err := ia.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ia
	}
	return res, nil
}

func (v *Validator) verifyAuditorSignature(signatureProvider driver.SignatureProvider) error {
	if v.pp.Auditor != nil {
		verifier, err := v.deserializer.GetAuditorVerifier(v.pp.Auditor)
		if err != nil {
			return errors.Errorf("failed to deserialize auditor's public key")
		}

		_, err = signatureProvider.HasBeenSignedBy(v.pp.Auditor, verifier)
		return err
	}
	return nil
}

func (v *Validator) verifyIssues(issues []driver.IssueAction, signatureProvider driver.SignatureProvider) error {
	for _, issue := range issues {
		a := issue.(*issue2.IssueAction)

		if err := v.verifyIssue(a); err != nil {
			return errors.Wrapf(err, "failed to verify issue action")
		}

		issuers := v.pp.Issuers
		if len(issuers) != 0 {
			// Check that a.Issuer is in issuers
			found := false
			for _, issuer := range issuers {
				if bytes.Equal(a.Issuer, issuer) {
					found = true
					break
				}
			}
			if !found {
				return errors.Errorf("issuer [%s] is not in issuers", view.Identity(a.Issuer).String())
			}
		}

		verifier, err := v.deserializer.GetIssuerVerifier(a.Issuer)
		if err != nil {
			return errors.Wrapf(err, "failed getting verifier for [%s]", view.Identity(a.Issuer).String())
		}
		if _, err := signatureProvider.HasBeenSignedBy(a.Issuer, verifier); err != nil {
			return errors.Wrapf(err, "failed verifying signature")
		}
	}
	return nil
}

func (v *Validator) verifyIssue(issue driver.IssueAction) error {
	action := issue.(*issue2.IssueAction)
	coms, err := action.GetCommitments()
	if err != nil {
		return errors.New("failed to verify issue")
	}
	return issue2.NewVerifier(
		coms,
		action.IsAnonymous(),
		v.pp).Verify(action.GetProof())
}

func (v *Validator) verifyTransfers(ledger driver.Ledger, transferActions []driver.TransferAction, signatureProvider driver.SignatureProvider) error {
	logger.Debugf("check sender start...")
	defer logger.Debugf("check sender finished.")
	for _, t := range transferActions {
		if err := v.verifyTransfer(t, ledger, signatureProvider); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action")
		}
	}
	return nil
}

func (v *Validator) verifyTransfer(tr driver.TransferAction, ledger driver.Ledger, signatureProvider driver.SignatureProvider) error {
	action := tr.(*transfer.TransferAction)
	context := &Context{
		PP:                v.pp,
		Deserializer:      v.deserializer,
		Action:            action,
		Ledger:            ledger,
		SignatureProvider: signatureProvider,
		MetadataCounter:   map[string]int{},
	}
	for _, v := range v.transferValidators {
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
