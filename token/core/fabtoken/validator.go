/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/interop"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var defaultValidators = []ValidateTransfer{SerializedIdentityTypeExtraValidator, ScriptTypeExchangeExtraValidator}

type ValidateTransfer func(inputTokens []*token2.UnspentToken, tr driver.TransferAction) error

func SerializedIdentityTypeExtraValidator(inputTokens []*token2.UnspentToken, tr driver.TransferAction) error {
	// noting else to validate
	return nil
}

func ScriptTypeExchangeExtraValidator(inputTokens []*token2.UnspentToken, tr driver.TransferAction) error {
	for _, in := range inputTokens {
		owner, err := identity.UnmarshallRawOwner(in.Owner.Raw)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal owner of input token")
		}
		if owner.Type == exchange.ScriptTypeExchange {
			if len(inputTokens) != 1 || len(tr.GetOutputs()) != 1 {
				return errors.Errorf("invalid transfer action: an exchange script only transfers the ownership of a token")
			}

			out := tr.GetOutputs()[0].(*TransferOutput).Output
			if inputTokens[0].Type != out.Type {
				return errors.Errorf("invalid transfer action: type of input does not match type of output")
			}
			if inputTokens[0].Quantity != out.Quantity {
				return errors.Errorf("invalid transfer action: quantity of input does not match quantity of output")
			}

			// check that owner field in output is correct
			if err := interop.VerifyTransferFromExchangeScript(inputTokens[0].Owner.Raw, out.Owner.Raw); err != nil {
				return errors.Wrap(err, "failed to verify transfer from exchange script")
			}
		}
	}

	for _, o := range tr.GetOutputs() {
		out, ok := o.(*TransferOutput)
		if !ok {
			return errors.Errorf("invalid output")
		}
		if out.IsRedeem() {
			continue
		}
		owner, err := identity.UnmarshallRawOwner(out.Output.Owner.Raw)
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

// Validator checks the validity of fabtoken TokenRequest
type Validator struct {
	// fabtoken public parameters
	pp *PublicParams
	// deserializer for identities used in fabtoken
	deserializer driver.Deserializer
	// extraValidators for performing additional validation
	extraValidators []ValidateTransfer
}

// NewValidator initializes a Validator with the passed parameters
func NewValidator(pp *PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransfer) (*Validator, error) {
	if pp == nil {
		return nil, errors.New("please provide a non-nil public parameters")
	}
	if deserializer == nil {
		return nil, errors.New("please provide a non-nil deserializer")
	}
	defaultValidators = append(defaultValidators, extraValidators...)
	return &Validator{
		pp:              pp,
		deserializer:    deserializer,
		extraValidators: defaultValidators,
	}, nil
}

// VerifyTokenRequest validates the passed token request against data in the ledger, the signature provided and the binding
func (v *Validator) VerifyTokenRequest(ledger driver.Ledger, signatureProvider driver.SignatureProvider, binding string, tr *driver.TokenRequest) ([]interface{}, error) {
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
	// audit is enabled
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

// VerifyAuditorSignature checks if the content of the token request concatenated with the binding
// was signed by the authorized auditor
func (v *Validator) VerifyAuditorSignature(signatureProvider driver.SignatureProvider) error {
	if v.pp.AuditorIdentity() != nil {
		verifier, err := v.deserializer.GetAuditorVerifier(v.pp.AuditorIdentity())
		if err != nil {
			return errors.New("failed to deserialize auditor's public key")
		}

		return signatureProvider.HasBeenSignedBy(v.pp.AuditorIdentity(), verifier)
	}
	return nil
}

// VerifyIssues checks if the issued tokens are valid and if the content of the token request concatenated
// with the binding was signed by one of the authorized issuers
func (v *Validator) VerifyIssues(issues []*IssueAction, signatureProvider driver.SignatureProvider) error {
	for _, issue := range issues {
		// verify that issue is valid
		if err := v.verifyIssue(issue); err != nil {
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
		if err := signatureProvider.HasBeenSignedBy(issue.Issuer, verifier); err != nil {
			return errors.Wrapf(err, "failed verifying signature")
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
		// verify if the token request concatenated with the binding was signed by the owners of the inputs
		// in the current transfer action
		err = v.CheckSendersSignatures(inputTokens, i, signatureProvider)
		if err != nil {
			return err
		}
		// verify if input tokens and output tokens in the current transfer action have the same type
		// verify if sum of input tokens  in the current transfer action equals the sum of output tokens
		// in the current transfer action
		if err := v.VerifyTransfer(inputTokens, t); err != nil {
			return errors.Wrapf(err, "failed to verify transfer action at index %d", i)
		}
	}
	return nil
}

// verifyIssue checks if all outputs in IssueAction are valid (no zero-value outputs)
func (v *Validator) verifyIssue(issue driver.IssueAction) error {
	if issue.NumOutputs() == 0 {
		return errors.Errorf("there is no output")
	}
	for _, output := range issue.GetOutputs() {
		out := output.(*TransferOutput).Output
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

// VerifyTransfer checks that sum of inputTokens in TransferAction equals sum of outputs in TransferAction
// It also checks that all outputs and inputs have the same type
func (v *Validator) VerifyTransfer(inputTokens []*token2.UnspentToken, tr driver.TransferAction) error {
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
	inputSum := token2.NewZeroQuantity(v.pp.QuantityPrecision)
	outputSum := token2.NewZeroQuantity(v.pp.QuantityPrecision)
	for i, input := range inputTokens {
		if input == nil {
			return errors.Errorf("input %d is nil", i)
		}
		q, err := token2.ToQuantity(input.Quantity, v.pp.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", input.Quantity)
		}
		inputSum.Add(q)
		// check that all inputs have the same type
		if input.Type != typ {
			return errors.Errorf("input type %s does not match type %s", input.Type, typ)
		}
	}
	for _, output := range tr.GetOutputs() {
		out := output.(*TransferOutput).Output
		q, err := token2.ToQuantity(out.Quantity, v.pp.QuantityPrecision)
		if err != nil {
			return errors.Wrapf(err, "failed parsing quantity [%s]", out.Quantity)
		}
		outputSum.Add(q)
		// check that all outputs have the same type and it is the same type as inputs
		if out.Type != typ {
			return errors.Errorf("output type %s does not match type %s", out.Type, typ)
		}
	}
	// check equality of sum of inputs and outputs
	if inputSum.Cmp(outputSum) != 0 {
		return errors.Errorf("input sum %v does not match output sum %v", inputSum, outputSum)
	}
	for _, v := range v.extraValidators {
		if err := v(inputTokens, tr); err != nil {
			return err
		}
	}
	return nil
}

type backend struct {
	getState driver.GetStateFnc
	// signed message
	message []byte
	index   int
	// signatures on message
	signatures [][]byte
}

// HasBeenSignedBy checks if a given message has been signed by the signing identity matching
// the passed verifier
// todo shall we remove id from the parameters
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

// UnmarshalIssueTransferActions returns the deserialized issue and transfer actions contained in the passed TokenRequest
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

// unmarshalTransferActions returns an array of deserialized TransferAction from raw bytes
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

// unmarshalIssueActions returns an array of deserialized IssueAction from raw bytes
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

// CheckSendersSignatures verifies if a TokenRequest was signed by the owners of the inputs in the TokenRequest
func (v *Validator) CheckSendersSignatures(inputTokens []*token2.UnspentToken, actionIndex int, signatureProvider driver.SignatureProvider) error {
	for _, tok := range inputTokens {
		logger.Debugf("check sender [%d][%s]", actionIndex, view.Identity(tok.Owner.Raw).UniqueID())
		verifier, err := v.deserializer.GetOwnerVerifierFromToken(&driver.UnspentToken{
			ID:    tok.Id,
			Owner: tok.Owner.Raw,
		})
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

// RetrieveInputsFromTransferAction retrieves from the passed ledger the inputs identified in TransferAction
func RetrieveInputsFromTransferAction(t *TransferAction, ledger driver.Ledger) ([]*token2.UnspentToken, error) {
	var inputTokens []*token2.UnspentToken
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
		id, err := keys.GetTokenIdFromKey(in)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract token id from input string [%s]", in)
		}
		inputTokens = append(inputTokens, &token2.UnspentToken{
			Id:       id,
			Owner:    tok.Owner,
			Type:     tok.Type,
			Quantity: tok.Quantity,
		})
	}
	return inputTokens, nil
}
