/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"bytes"
	"encoding/json"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/interop"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog")

var defaultValidators = []ValidateTransfer{SerializedIdentityTypeExtraValidator, ScriptTypeExchangeExtraValidator}

type ValidateTransfer func(tokens []*token.Token, tr *transfer.TransferAction) error

func SerializedIdentityTypeExtraValidator(tokens []*token.Token, tr *transfer.TransferAction) error {
	// noting else to validate
	return nil
}

func ScriptTypeExchangeExtraValidator(tokens []*token.Token, tr *transfer.TransferAction) error {
	for _, in := range tokens {
		owner, err := identity.UnmarshallRawOwner(in.Owner)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if owner.Type == exchange.ScriptTypeExchange {
			if len(tokens) != 1 || len(tr.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: an exchange script only transfers the ownership of a token")
			}

			out := tr.GetOutputs()[0].(*token.Token)
			if tokens[0].Data.Equals(out.Data) {
				return errors.Errorf("invalid transfer action: content of input does not match content of output")
			}

			// check that owner field in output is correct
			if err := interop.VerifyTransferFromExchangeScript(tokens[0].Owner, out.Owner); err != nil {
				return errors.Wrap(err, "failed to verify transfer from exchange script")
			}
		}
	}

	for _, o := range tr.GetOutputs() {
		out, ok := o.(*token.Token)
		if !ok {
			return errors.Errorf("invalid output")
		}
		if out.IsRedeem() {
			continue
		}
		owner, err := identity.UnmarshallRawOwner(out.Owner)
		if err != nil {
			return err
		}
		if owner.Type == exchange.ScriptTypeExchange {
			script := &exchange.Script{}
			err = json.Unmarshal(owner.Identity, script)
			if err != nil {
				return err
			}
			if script.Deadline.Before(time.Now()) {
				return errors.Errorf("exchange script invalid: expiration date has already passed")
			}
			continue
		}
	}
	return nil
}

type Validator struct {
	pp              *crypto.PublicParams
	deserializer    driver.Deserializer
	extraValidators []ValidateTransfer
}

func New(pp *crypto.PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransfer) *Validator {
	defaultValidators = append(defaultValidators, extraValidators...)
	return &Validator{
		pp:              pp,
		deserializer:    deserializer,
		extraValidators: defaultValidators,
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

	backend := &backend{
		getState:   getState,
		message:    signed,
		signatures: signatures,
	}
	return v.VerifyTokenRequest(backend, backend, binding, tr)
}

type Signature struct {
	metadata map[string][]byte // metadata may include for example the preimage of an exchange script
}

func (s *Signature) Metadata() map[string][]byte {
	return s.metadata
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

	for _, sig := range signatureProvider.Signatures() {
		claim := &exchange.ClaimSignature{}
		if err = json.Unmarshal(sig, claim); err != nil {
			continue
		}
		if len(claim.Preimage) == 0 || len(claim.RecipientSignature) == 0 {
			return nil, errors.New("expected a valid claim preImage and recipient signature")
		}
		actions = append(actions, &Signature{
			metadata: map[string][]byte{
				"claimPreimage": claim.Preimage,
			},
		})
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

		return signatureProvider.HasBeenSignedBy(v.pp.Auditor, verifier)
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
		if err := signatureProvider.HasBeenSignedBy(a.Issuer, verifier); err != nil {
			return errors.Wrapf(err, "failed verifying signature")
		}
	}
	return nil
}

func (v *Validator) verifyTransfers(ledger driver.Ledger, transferActions []driver.TransferAction, signatureProvider driver.SignatureProvider) error {
	logger.Debugf("check sender start...")
	defer logger.Debugf("check sender finished.")
	for i, t := range transferActions {
		var inputTokens [][]byte
		inputs, err := t.GetInputs()
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve inputs to spend")
		}
		for _, in := range inputs {
			logger.Debugf("load token [%d][%s]", i, in)
			bytes, err := ledger.GetState(in)
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve input to spend [%s]", in)
			}
			if len(bytes) == 0 {
				return errors.Errorf("input to spend [%s] does not exists", in)
			}
			inputTokens = append(inputTokens, bytes)
			tok := &token.Token{}
			err = tok.Deserialize(bytes)
			if err != nil {
				return errors.Wrapf(err, "failed to deserialize input to spend [%s]", in)
			}
			logger.Debugf("check sender [%d][%s]", i, view.Identity(tok.Owner).UniqueID())
			verifier, err := v.deserializer.GetOwnerVerifier(tok.Owner)
			if err != nil {
				return errors.Wrapf(err, "failed deserializing owner [%d][%s][%s]", i, in, view.Identity(tok.Owner).UniqueID())
			}
			logger.Debugf("signature verification [%d][%s][%s]", i, in, view.Identity(tok.Owner).UniqueID())
			if err := signatureProvider.HasBeenSignedBy(tok.Owner, verifier); err != nil {
				return errors.Wrapf(err, "failed signature verification [%d][%s][%s]", i, in, view.Identity(tok.Owner).UniqueID())
			}
		}
		if err := v.verifyTransfer(inputTokens, t); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action")
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

func (v *Validator) verifyTransfer(inputTokens [][]byte, tr driver.TransferAction) error {
	action := tr.(*transfer.TransferAction)
	tokens := make([]*token.Token, len(inputTokens))
	in := make([]*math.G1, len(inputTokens))
	for i, raw := range inputTokens {
		tokens[i] = &token.Token{}
		if err := tokens[i].Deserialize(raw); err != nil {
			return errors.Wrapf(err, "invalid transfer: failed to deserialize input [%d]", i)
		}
		in[i] = tokens[i].GetCommitment()
	}

	if err := transfer.NewVerifier(
		in,
		action.GetOutputCommitments(),
		v.pp).Verify(action.GetProof()); err != nil {
		return err
	}

	for _, v := range v.extraValidators {
		if err := v(tokens, action); err != nil {
			return err
		}
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
