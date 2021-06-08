/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package validator

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/anonym"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog")

type Validator struct {
	pp *crypto.PublicParams
}

func New(pp *crypto.PublicParams) *Validator {
	return &Validator{pp: pp}
}

func (v *Validator) VerifyTokenRequestFromRaw(getState api.GetStateFnc, binding string, raw []byte) ([]interface{}, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty token request")
	}
	tr := &api.TokenRequest{}
	err := json.Unmarshal(raw, tr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token request")
	}

	// Prepare message expected to be signed
	// TODO: encapsulate this somewhere
	req := &api.TokenRequest{}
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

func (v *Validator) VerifyTokenRequest(ledger api.Ledger, signatureProvider api.SignatureProvider, binding string, tr *api.TokenRequest) ([]interface{}, error) {
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

func (v *Validator) unmarshalTransferActions(raw [][]byte) ([]api.TransferAction, error) {
	res := make([]api.TransferAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ta := &transfer.TransferAction{}
		if err := ta.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ta
	}
	return res, nil
}

func (v *Validator) unmarshalIssueActions(raw [][]byte) ([]api.IssueAction, error) {
	res := make([]api.IssueAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ia := &issue2.IssueAction{}
		if err := ia.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ia
	}
	return res, nil
}

func (v *Validator) verifyAuditorSignature(signatureProvider api.SignatureProvider) error {
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

func (v *Validator) verifyIssues(issues []api.IssueAction, signatureProvider api.SignatureProvider) error {
	for _, issue := range issues {
		a := issue.(*issue2.IssueAction)

		if err := v.verifyIssue(a); err != nil {
			return errors.Wrapf(err, "failed to verify issue action")
		}

		if a.Anonymous {
			verifier := &anonym.Verifier{}
			ip := &crypto.IssuingPolicy{}
			err := ip.Deserialize(v.pp.IssuingPolicy)
			if err != nil {
				return err
			}
			err = verifier.Deserialize(ip.BitLength, ip.Issuers, v.pp.ZKATPedParams, a.OutputTokens[0].Data, a.Issuer)
			if err != nil {
				return err
			}
			if err := signatureProvider.HasBeenSignedBy(a.Issuer, verifier); err != nil {
				return errors.Wrapf(err, "failed verifying signature")
			}
		} else {
			identityDeserializer := &fabric.MSPX509IdentityDeserializer{}
			verifier, err := identityDeserializer.GetVerifier(a.Issuer)
			if err != nil {
				return errors.Wrapf(err, "failed getting verifier for [%s]", view.Identity(a.Issuer).String())
			}
			if err := signatureProvider.HasBeenSignedBy(a.Issuer, verifier); err != nil {
				return errors.Wrapf(err, "failed verifying signature")
			}
		}
	}
	return nil
}

func (v *Validator) verifyTransfers(ledger api.Ledger, transferActions []api.TransferAction, signatureProvider api.SignatureProvider) error {
	identityDeserializer, err := idemix2.NewDeserializer(v.pp.IdemixPK)
	if err != nil {
		return errors.Wrap(err, "failed instantiating deserializer")
	}

	logger.Debugf("check sender start...")
	defer logger.Debugf("check sender finished.")
	for i, t := range transferActions {
		var inputTokens [][]byte
		inputs, err := t.GetInputs()
		if err != nil {
			errors.Wrapf(err, "failed to retrieve inputs to spend")
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
			verifier, err := identityDeserializer.DeserializeVerifier(tok.Owner)
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

func (v *Validator) verifyIssue(issue api.IssueAction) error {
	action := issue.(*issue2.IssueAction)

	return issue2.NewVerifier(
		action.GetCommitments(),
		action.IsAnonymous(),
		v.pp).Verify(action.GetProof())
}

func (v *Validator) verifyTransfer(inputTokens [][]byte, tr api.TransferAction) error {
	action := tr.(*transfer.TransferAction)

	in := make([]*bn256.G1, len(inputTokens))
	for i, raw := range inputTokens {
		tok := &token.Token{}
		if err := tok.Deserialize(raw); err != nil {
			return errors.Wrapf(err, "invalid transfer: failed to deserialize input [%d]", i)
		}
		in[i] = tok.GetCommitment()
	}

	return transfer.NewVerifier(
		in,
		action.GetOutputCommitments(),
		v.pp).Verify(action.GetProof())
}

type backend struct {
	getState   api.GetStateFnc
	message    []byte
	index      int
	signatures [][]byte
}

func (b *backend) HasBeenSignedBy(id view.Identity, verifier api.Verifier) error {
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
