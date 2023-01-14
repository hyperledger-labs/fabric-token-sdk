/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"encoding/asn1"

	"go.uber.org/zap/zapcore"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	TransferMetadataPrefix = "TransferMetadataPrefix"
)

// IssueOptions models the options that can be passed to the issue command
type IssueOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
}

func compileIssueOptions(opts ...IssueOption) (*IssueOptions, error) {
	txOptions := &IssueOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

// IssueOption is a function that modify IssueOptions
type IssueOption func(*IssueOptions) error

// WithIssueAttribute sets an attribute to be used to customize the issue command
func WithIssueAttribute(attr, value interface{}) IssueOption {
	return func(o *IssueOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes[attr] = value
		return nil
	}
}

// TransferOptions models the options that can be passed to the transfer command
type TransferOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
	// Selector is the custom token selector to use. If nil, the default will be used.
	Selector Selector
	// TokenIDs to transfer. If empty, the tokens will be selected.
	TokenIDs []*token.ID
}

func compileTransferOptions(opts ...TransferOption) (*TransferOptions, error) {
	txOptions := &TransferOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

// TransferOption is a function that modify TransferOptions
type TransferOption func(*TransferOptions) error

// WithTokenSelector sets the passed token selector
func WithTokenSelector(selector Selector) TransferOption {
	return func(o *TransferOptions) error {
		o.Selector = selector
		return nil
	}
}

// WithTransferMetadata sets transfer action metadata
func WithTransferMetadata(key string, value []byte) TransferOption {
	return WithTransferAttribute(TransferMetadataPrefix+key, value)
}

// WithTokenIDs sets the tokens ids to transfer
func WithTokenIDs(ids ...*token.ID) TransferOption {
	return func(o *TransferOptions) error {
		o.TokenIDs = ids
		return nil
	}
}

// WithTransferAttribute sets an attribute to be used to customize the transfer command
func WithTransferAttribute(attr, value interface{}) TransferOption {
	return func(o *TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = make(map[interface{}]interface{})
		}
		o.Attributes[attr] = value
		return nil
	}
}

// AuditRecord models the audit record returned by the audit command
// It contains the token request's anchor, inputs (with Type and Quantity), and outputs
type AuditRecord struct {
	Anchor  string
	Inputs  *InputStream
	Outputs *OutputStream
}

// Issue contains information about an issue operation.
// In particular, it carries the identities of the issuer and the receivers
type Issue struct {
	Issuer    view.Identity
	Receivers []view.Identity
}

// Transfer contains information about a transfer operation.
// In particular, it carries the identities of the senders and the receivers
type Transfer struct {
	// Senders is the list of identities of the senders
	Senders []view.Identity
	// Receivers is the list of identities of the receivers
	Receivers []view.Identity
	// ExtraSigners is the list of extra identities that must sign the token request to make it valid.
	// This field is to be used by the token drivers to list any additional identities that must
	// sign the token request.
	ExtraSigners []view.Identity
}

// Request aggregates token operations that must be performed atomically.
// Operations are represented in a backend agnostic way but driver specific.
type Request struct {
	// Anchor is used to bind the Actions to a given Transaction
	Anchor string
	// Actions contains the token operations.
	Actions *driver.TokenRequest
	// Metadata contains the actions' metadata used to unscramble the content of the actions, if the
	// underlying token driver requires that
	Metadata *driver.TokenRequestMetadata
	// TokenService this request refers to
	TokenService *ManagementService `json:"-"`
}

// NewRequest creates a new empty request for the given token service and anchor
func NewRequest(tokenService *ManagementService, anchor string) *Request {
	return &Request{
		Anchor:       anchor,
		Actions:      &driver.TokenRequest{},
		Metadata:     &driver.TokenRequestMetadata{},
		TokenService: tokenService,
	}
}

// NewRequestFromBytes creates a new request from the given anchor, and whose actions and metadata
// are unmarshalled from the given bytes
func NewRequestFromBytes(tokenService *ManagementService, anchor string, actions []byte, trmRaw []byte) (*Request, error) {
	tr := &driver.TokenRequest{}
	if err := tr.FromBytes(actions); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling token request [%d]", len(actions))
	}
	trm := &driver.TokenRequestMetadata{}
	if len(trmRaw) != 0 {
		if err := trm.FromBytes(trmRaw); err != nil {
			return nil, errors.Wrapf(err, "failed unmarshalling token request metadata [%d]", len(trmRaw))
		}
	}
	return &Request{
		Anchor:       anchor,
		Actions:      tr,
		Metadata:     trm,
		TokenService: tokenService,
	}, nil
}

// ID returns the anchor of the request
func (r *Request) ID() string {
	return r.Anchor
}

// Issue appends an issue action to the request. The action will be prepared using the provided issuer wallet.
// The action issues to the receiver a token of the passed type and quantity.
// Additional options can be passed to customize the action.
func (r *Request) Issue(wallet *IssuerWallet, receiver view.Identity, typ string, q uint64, opts ...IssueOption) (*IssueAction, error) {
	if wallet == nil {
		return nil, errors.Errorf("wallet is nil")
	}
	if typ == "" {
		return nil, errors.Errorf("type is empty")
	}
	if q == 0 {
		return nil, errors.Errorf("q is zero")
	}
	maxTokenValue := r.TokenService.PublicParametersManager().MaxTokenValue()
	if q > maxTokenValue {
		return nil, errors.Errorf("q is larger than max token value [%d]", maxTokenValue)
	}

	if receiver.IsNone() {
		return nil, errors.Errorf("all recipients should be defined")
	}

	id, err := wallet.GetIssuerIdentity(typ)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting issuer identity for type [%s]", typ)
	}

	opt, err := compileIssueOptions(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed compiling options [%v]", opts)
	}

	// Compute Issue
	issue, tokenInfos, issuer, err := r.TokenService.tms.Issue(
		id,
		typ,
		[]uint64{q},
		[][]byte{receiver},
		&driver.IssueOptions{
			Attributes: opt.Attributes,
		},
	)
	if err != nil {
		return nil, err
	}

	// Append
	raw, err := issue.Serialize()
	if err != nil {
		return nil, err
	}
	r.Actions.Issues = append(r.Actions.Issues, raw)
	outputs, err := issue.GetSerializedOutputs()
	if err != nil {
		return nil, err
	}

	auditInfo, err := r.TokenService.tms.GetAuditInfo(receiver)
	if err != nil {
		return nil, err
	}
	if r.Metadata == nil {
		return nil, errors.New("failed to complete issue: nil Metadata in token request")
	}
	r.Metadata.Issues = append(r.Metadata.Issues,
		driver.IssueMetadata{
			Issuer:              issuer,
			Outputs:             outputs,
			TokenInfo:           tokenInfos,
			Receivers:           []view.Identity{receiver},
			ReceiversAuditInfos: [][]byte{auditInfo},
		},
	)

	return &IssueAction{a: issue}, nil
}

// Transfer appends a transfer action to the request. The action will be prepared using the provided owner wallet.
// The action transfers tokens of the passed types to the receivers for the passed quantities.
// In other words, owners[0] will receives values[0], and so on.
// Additional options can be passed to customize the action.
func (r *Request) Transfer(wallet *OwnerWallet, typ string, values []uint64, owners []view.Identity, opts ...TransferOption) (*TransferAction, error) {
	for _, v := range values {
		if v == 0 {
			return nil, errors.Errorf("value is zero")
		}
	}
	opt, err := compileTransferOptions(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed compiling options [%v]", opts)
	}
	tokenIDs, outputTokens, err := r.prepareTransfer(false, wallet, typ, values, owners, opt)
	if err != nil {
		return nil, errors.Wrap(err, "failed preparing transfer")
	}

	logger.Debugf("Prepare Transfer Action [id:%s,ins:%d,outs:%d]", r.Anchor, len(tokenIDs), len(outputTokens))

	ts := r.TokenService.tms

	// Compute transfer
	transfer, transferMetadata, err := ts.Transfer(
		r.Anchor,
		wallet.w,
		tokenIDs,
		outputTokens,
		&driver.TransferOptions{
			Attributes: opt.Attributes,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating transfer action")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		// double check
		if err := ts.VerifyTransfer(transfer, transferMetadata.OutputsMetadata); err != nil {
			return nil, errors.Wrap(err, "failed checking generated proof")
		}
	}

	// Append
	raw, err := transfer.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing transfer action")
	}
	r.Actions.Transfers = append(r.Actions.Transfers, raw)
	r.Metadata.Transfers = append(r.Metadata.Transfers, *transferMetadata)

	return &TransferAction{a: transfer}, nil
}

// Redeem appends a redeem action to the request. The action will be prepared using the provided owner wallet.
// The action redeems tokens of the passed type for a total amount matching the passed value.
// Additional options can be passed to customize the action.
func (r *Request) Redeem(wallet *OwnerWallet, typ string, value uint64, opts ...TransferOption) error {
	opt, err := compileTransferOptions(opts...)
	if err != nil {
		return errors.WithMessagef(err, "failed compiling options [%v]", opts)
	}
	tokenIDs, outputTokens, err := r.prepareTransfer(true, wallet, typ, []uint64{value}, []view.Identity{nil}, opt)
	if err != nil {
		return errors.Wrap(err, "failed preparing transfer")
	}

	logger.Debugf("Prepare Redeem Action [ins:%d,outs:%d]", len(tokenIDs), len(outputTokens))

	ts := r.TokenService.tms

	// Compute redeem, it is a transfer with owner set to nil
	transfer, transferMetadata, err := ts.Transfer(
		r.Anchor,
		wallet.w,
		tokenIDs,
		outputTokens,
		&driver.TransferOptions{
			Attributes: opt.Attributes,
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed creating transfer action")
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		// double check
		if err := ts.VerifyTransfer(transfer, transferMetadata.OutputsMetadata); err != nil {
			return errors.Wrap(err, "failed checking generated proof")
		}
	}

	// Append
	raw, err := transfer.Serialize()
	if err != nil {
		return errors.Wrap(err, "failed serializing transfer action")
	}

	r.Actions.Transfers = append(r.Actions.Transfers, raw)
	r.Metadata.Transfers = append(r.Metadata.Transfers, *transferMetadata)

	return nil
}

// Outputs returns the sequence of outputs of the request supporting sequential and parallel aggregate operations.
func (r *Request) Outputs() (*OutputStream, error) {
	return r.outputs(false)
}

func (r *Request) outputs(failOnMissing bool) (*OutputStream, error) {
	tms := r.TokenService.tms
	meta, err := r.GetMetadata()
	if err != nil {
		return nil, err
	}
	var outputs []*Output
	counter := uint64(0)
	for i, issue := range r.Actions.Issues {
		// deserialize action
		issueAction, err := tms.DeserializeIssueAction(issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing issue action [%d]", i)
		}
		// get metadata for action
		issueMeta, err := meta.Issue(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting issue metadata [%d]", i)
		}
		if err := issueMeta.Match(&IssueAction{a: issueAction}); err != nil {
			return nil, errors.Wrapf(err, "failed matching issue action with its metadata [%d]", i)
		}

		extractedOutputs, err := r.extractIssueOutputs(i, counter, issueAction, issueMeta, failOnMissing)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter += uint64(len(outputs))
	}

	for i, transfer := range r.Actions.Transfers {
		// deserialize action
		transferAction, err := tms.DeserializeTransferAction(transfer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing transfer action [%d]", i)
		}
		// get metadata for action
		transferMeta, err := meta.Transfer(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting transfer metadata [%d]", i)
		}
		if err := transferMeta.Match(&TransferAction{a: transferAction}); err != nil {
			return nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]", i)
		}
		if len(transferAction.GetOutputs()) != len(transferMeta.OutputsMetadata) || len(transferMeta.ReceiverAuditInfos) != len(transferAction.GetOutputs()) {
			return nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]: invalid metadata", i)
		}

		extractedOutputs, err := r.extractTransferOutputs(i, counter, transferAction, transferMeta, failOnMissing)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter += uint64(len(extractedOutputs))
	}

	return NewOutputStream(outputs, tms.PublicParamsManager().PublicParameters().Precision()), nil
}

func (r *Request) extractIssueOutputs(i int, counter uint64, issueAction driver.IssueAction, issueMeta *IssueMetadata, failOnMissing bool) ([]*Output, error) {
	// extract outputs for this action
	tms := r.TokenService.tms
	precision := tms.PublicParamsManager().PublicParameters().Precision()
	var outputs []*Output
	for j, output := range issueAction.GetOutputs() {
		if output == nil {
			return nil, errors.Errorf("%d^th output in issue action [%d] is nil", j, i)
		}
		raw, err := output.Serialize()
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing issue action output [%d,%d]", i, j)
		}

		// is the j-th meta present? It might have been filtered out
		if issueMeta.IsOutputAbsent(j) {
			logger.Debugf("Issue Action Output [%d,%d] is absent", i, j)
			if failOnMissing {
				return nil, errors.Errorf("missing token info for output [%d,%d]", i, j)
			}
			continue
		}
		if len(issueAction.GetOutputs()) != len(issueMeta.TokenInfo) || len(issueMeta.ReceiversAuditInfos) != len(issueAction.GetOutputs()) {
			return nil, errors.Wrapf(err, "failed matching issue action with its metadata [%d]: invalid metadata", i)
		}
		tok, _, err := tms.DeserializeToken(raw, issueMeta.TokenInfo[j])
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting issue action output in the clear [%d,%d]", i, j)
		}
		eID, err := tms.GetEnrollmentID(issueMeta.ReceiversAuditInfos[j])
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting enrollment id [%d,%d]", i, j)
		}
		q, err := token.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting quantity [%d,%d]", i, j)
		}

		outputs = append(outputs, &Output{
			ActionIndex:    i,
			Index:          counter,
			Owner:          tok.Owner.Raw,
			OwnerAuditInfo: issueMeta.ReceiversAuditInfos[j],
			EnrollmentID:   eID,
			Type:           tok.Type,
			Quantity:       q,
		})
		counter++
	}
	return outputs, nil
}

func (r *Request) extractTransferOutputs(i int, counter uint64, transferAction driver.TransferAction, transferMeta *TransferMetadata, failOnMissing bool) ([]*Output, error) {
	tms := r.TokenService.tms
	if tms.PublicParamsManager() == nil || tms.PublicParamsManager().PublicParameters() == nil {
		return nil, errors.New("can't get inputs: invalid token service in request")
	}
	precision := tms.PublicParamsManager().PublicParameters().Precision()
	var outputs []*Output
	for j, output := range transferAction.GetOutputs() {
		if output == nil {
			return nil, errors.Errorf("%d^th output in transfer action [%d] is nil", j, i)
		}
		raw, err := output.Serialize()
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing transfer action output [%d,%d]", i, j)
		}

		// is the j-th meta present? It might have been filtered out
		if transferMeta.IsOutputAbsent(j) {
			logger.Debugf("Transfer Action Output [%d,%d] is absent", i, j)
			if failOnMissing {
				return nil, errors.Errorf("missing token info for output [%d,%d]", i, j)
			}
			continue
		}

		tok, _, err := tms.DeserializeToken(raw, transferMeta.OutputsMetadata[j])
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting transfer action output in the clear [%d,%d]", i, j)
		}
		var eID string
		var ownerAuditInfo []byte
		if len(tok.Owner.Raw) != 0 {
			ownerAuditInfo = transferMeta.ReceiverAuditInfos[j]
			eID, err = tms.GetEnrollmentID(ownerAuditInfo)
			if err != nil {
				return nil, errors.Wrapf(err, "failed getting enrollment id [%d,%d]", i, j)
			}
		}

		q, err := token.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting quantity [%d,%d]", i, j)
		}

		outputs = append(outputs, &Output{
			ActionIndex:    i,
			Index:          counter,
			Owner:          tok.Owner.Raw,
			OwnerAuditInfo: ownerAuditInfo,
			EnrollmentID:   eID,
			Type:           tok.Type,
			Quantity:       q,
		})
		counter++
	}

	return outputs, nil
}

// Inputs returns the sequence of inputs of the request supporting sequential and parallel aggregate operations.
// Notice that the inputs do not carry Type and Quantity because this information might be available to all parties.
// If you are an auditor, you can use the AuditInputs method to get everything.
func (r *Request) Inputs() (*InputStream, error) {
	return r.inputs(false)
}

func (r *Request) inputs(failOnMissing bool) (*InputStream, error) {
	tms := r.TokenService.tms
	if tms.PublicParamsManager() == nil || tms.PublicParamsManager().PublicParameters() == nil {
		return nil, errors.New("can't get inputs: invalid token service in request")
	}
	meta, err := r.GetMetadata()
	if err != nil {
		return nil, err
	}
	var inputs []*Input
	for i, transfer := range r.Actions.Transfers {
		// deserialize action
		transferAction, err := tms.DeserializeTransferAction(transfer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing transfer action [%d]", i)
		}
		// get metadata for action
		transferMeta, err := meta.Transfer(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting transfer metadata [%d]", i)
		}
		if err := transferMeta.Match(&TransferAction{a: transferAction}); err != nil {
			return nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]", i)
		}

		// we might not have TokenIDs if they have been filtered
		if len(transferMeta.TokenIDs) == 0 && failOnMissing {
			return nil, errors.Errorf("missing token ids for transfer [%d]", i)
		}

		extractedInputs, err := r.extractInputs(i, transferMeta, failOnMissing)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, extractedInputs...)
	}
	return NewInputStream(r.TokenService.Vault().NewQueryEngine(), inputs, tms.PublicParamsManager().PublicParameters().Precision()), nil
}

func (r *Request) extractInputs(i int, transferMeta *TransferMetadata, failOnMissing bool) ([]*Input, error) {
	// Iterate over the transferMeta.SenderAuditInfos because we know that there will be at least one
	// sender, but it might be that there are not token IDs due to filtering.
	tms := r.TokenService.tms
	var inputs []*Input
	for j, senderAuditInfo := range transferMeta.SenderAuditInfos {
		// The recipient might be missing because it has been filtered out. Skip in this case
		if transferMeta.IsInputAbsent(j) {
			if failOnMissing {
				return nil, errors.Errorf("missing receiver for transfer [%d,%d]", i, j)
			}
			continue
		}

		eID, err := tms.GetEnrollmentID(senderAuditInfo)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting enrollment id [%d,%d]", i, j)
		}

		inputs = append(inputs, &Input{
			ActionIndex:    i,
			Id:             transferMeta.TokenIDAt(j),
			Owner:          transferMeta.Senders[j],
			OwnerAuditInfo: senderAuditInfo,
			EnrollmentID:   eID,
		})
	}
	return inputs, nil
}

func (r *Request) InputsAndOutputs() (*InputStream, *OutputStream, error) {
	return r.inputsAndOutputs(false, false)
}

func (r *Request) inputsAndOutputs(failOnMissing, verifyActions bool) (*InputStream, *OutputStream, error) {
	tms := r.TokenService.tms
	if tms.PublicParamsManager() == nil || tms.PublicParamsManager().PublicParameters() == nil {
		return nil, nil, errors.New("can't get inputs: invalid token service in request")
	}
	meta, err := r.GetMetadata()
	if err != nil {
		return nil, nil, err
	}
	var inputs []*Input
	var outputs []*Output
	counter := uint64(0)

	for i, issue := range r.Actions.Issues {
		// deserialize action
		issueAction, err := tms.DeserializeIssueAction(issue)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed deserializing issue action [%d]", i)
		}
		// get metadata for action
		issueMeta, err := meta.Issue(i)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting issue metadata [%d]", i)
		}
		if err := issueMeta.Match(&IssueAction{a: issueAction}); err != nil {
			return nil, nil, errors.Wrapf(err, "failed matching issue action with its metadata [%d]", i)
		}

		if verifyActions {
			if err := tms.VerifyIssue(issueAction, issueMeta.TokenInfo); err != nil {
				return nil, nil, errors.WithMessagef(err, "failed verifying issue action")
			}
		}

		extractedOutputs, err := r.extractIssueOutputs(i, counter, issueAction, issueMeta, failOnMissing)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter += uint64(len(outputs))
	}

	for i, transfer := range r.Actions.Transfers {
		// deserialize action
		transferAction, err := tms.DeserializeTransferAction(transfer)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed deserializing transfer action [%d]", i)
		}
		// get metadata for action
		transferMeta, err := meta.Transfer(i)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting transfer metadata [%d]", i)
		}
		if err := transferMeta.Match(&TransferAction{a: transferAction}); err != nil {
			return nil, nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]", i)
		}
		if verifyActions {
			if err := tms.VerifyTransfer(transferAction, transferMeta.OutputsMetadata); err != nil {
				return nil, nil, errors.WithMessagef(err, "failed verifying transfer action")
			}
		}

		// we might not have TokenIDs if they have been filtered
		if len(transferMeta.TokenIDs) == 0 && failOnMissing {
			return nil, nil, errors.Errorf("missing token ids for transfer [%d]", i)
		}

		extractedInputs, err := r.extractInputs(i, transferMeta, failOnMissing)
		if err != nil {
			return nil, nil, err
		}
		inputs = append(inputs, extractedInputs...)

		extractedOutputs, err := r.extractTransferOutputs(i, counter, transferAction, transferMeta, failOnMissing)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter += uint64(len(extractedOutputs))
	}

	precision := tms.PublicParamsManager().PublicParameters().Precision()
	is := NewInputStream(r.TokenService.Vault().NewQueryEngine(), inputs, precision)
	os := NewOutputStream(outputs, precision)
	return is, os, nil
}

// IsValid checks that the request is valid.
func (r *Request) IsValid() error {
	// check request fields
	if r.TokenService == nil {
		return errors.New("invalid token service in request")
	}
	if r.Actions == nil {
		return errors.New("invalid actions in request")
	}
	if r.Metadata == nil {
		return errors.New("invalid metadata in request")
	}

	// check inputs, outputs, and verify actions
	if _, _, err := r.inputsAndOutputs(false, true); err != nil {
		return errors.WithMessagef(err, "failed verifying inputs and outputs")
	}

	return nil
}

// MarshalToAudit marshals the request to a message suitable for audit signature.
// In particular, metadata is not included.
func (r *Request) MarshalToAudit() ([]byte, error) {
	if r.Actions == nil {
		return nil, errors.Errorf("failed to marshal request in tx [%s] for audit", r.Anchor)
	}
	bytes, err := asn1.Marshal(driver.TokenRequest{Issues: r.Actions.Issues, Transfers: r.Actions.Transfers})
	if err != nil {
		return nil, errors.Wrapf(err, "audit of tx [%s] failed: error marshal token request for signature", r.Anchor)
	}
	return append(bytes, []byte(r.Anchor)...), nil
}

// MarshalToSign marshals the request to a message suitable for signing.
func (r *Request) MarshalToSign() ([]byte, error) {
	if r.Actions == nil {
		return nil, errors.Errorf("failed to marshal request in tx [%s] for signing", r.Anchor)
	}
	return r.TokenService.tms.MarshalTokenRequestToSign(r.Actions, r.Metadata)
}

// RequestToBytes marshals the request's actions to bytes.
func (r *Request) RequestToBytes() ([]byte, error) {
	if r.Actions == nil {
		return nil, errors.Errorf("failed to marshal request in tx [%s]", r.Anchor)
	}
	return r.Actions.Bytes()
}

// MetadataToBytes marshals the request's metadata to bytes.
func (r *Request) MetadataToBytes() ([]byte, error) {
	if r.Metadata == nil {
		return nil, errors.Errorf("failed to marshal metadata for request in tx [%s]", r.Anchor)
	}
	return r.Metadata.Bytes()
}

// Bytes marshals the request to bytes.
// It includes: Anchor (or ID), actions, and metadata.
func (r *Request) Bytes() ([]byte, error) {
	req, err := r.RequestToBytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling request to bytes")
	}
	meta, err := r.MetadataToBytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling metadata to bytes")
	}
	return asn1.Marshal(requestSer{
		TxID:     r.Anchor,
		Actions:  req,
		Metadata: meta,
	})
}

// FromBytes unmarshalls the request from bytes overriding the content of the current request.
func (r *Request) FromBytes(request []byte) error {
	var req requestSer
	_, err := asn1.Unmarshal(request, &req)
	if err != nil {
		return errors.Wrapf(err, "failed unmarshalling request")
	}
	r.Anchor = req.TxID
	if len(req.Actions) > 0 {
		if err := r.Actions.FromBytes(req.Actions); err != nil {
			return errors.Wrapf(err, "failed unmarshalling actions")
		}
	}
	if len(req.Metadata) > 0 {
		if err := r.Metadata.FromBytes(req.Metadata); err != nil {
			return errors.Wrapf(err, "failed unmarshalling metadata")
		}
	}
	return nil
}

// AddAuditorSignature adds an auditor signature to the request.
func (r *Request) AddAuditorSignature(sigma []byte) {
	r.Actions.AuditorSignatures = append(r.Actions.AuditorSignatures, sigma)
}

// AppendSignature appends a signature to the request.
func (r *Request) AppendSignature(sigma []byte) {
	r.Actions.Signatures = append(r.Actions.Signatures, sigma)
}

// SetTokenService sets the token service.
func (r *Request) SetTokenService(service *ManagementService) {
	r.TokenService = service
}

// BindTo binds transfers' senders and receivers, that are senders, that are not me to the passed identity
func (r *Request) BindTo(sp view2.ServiceProvider, party view.Identity) error {
	resolver := view2.GetEndpointService(sp)
	longTermIdentity, _, _, err := view2.GetEndpointService(sp).Resolve(party)
	if err != nil {
		return errors.Wrap(err, "cannot resolve identity")
	}

	for i := range r.Actions.Transfers {
		// senders
		for _, eid := range r.Metadata.Transfers[i].Senders {
			if w := r.TokenService.WalletManager().Wallet(eid); w != nil {
				// this is me, skip
				continue
			}
			logger.Debugf("bind sender [%s] to [%s]", eid, party)
			if err := resolver.Bind(longTermIdentity, eid); err != nil {
				return errors.Wrap(err, "failed binding sender identities")
			}
		}

		// extra signers
		for _, eid := range r.Metadata.Transfers[i].ExtraSigners {
			if w := r.TokenService.WalletManager().Wallet(eid); w != nil {
				// this is me, skip
				continue
			}
			logger.Debugf("bind extra sginer [%s] to [%s]", eid, party)
			if err := resolver.Bind(longTermIdentity, eid); err != nil {
				return errors.Wrap(err, "failed binding sender identities")
			}
		}

		// receivers
		receivers := r.Metadata.Transfers[i].Receivers
		for j, b := range r.Metadata.Transfers[i].ReceiverIsSender {
			if b {
				if w := r.TokenService.WalletManager().Wallet(receivers[j]); w != nil {
					// this is me, skip
					continue
				}

				logger.Debugf("bind receiver as sender [%s] to [%s]", receivers[j], party)
				if err := resolver.Bind(longTermIdentity, receivers[j]); err != nil {
					return errors.Wrap(err, "failed binding receiver identities")
				}
			}
		}
	}
	return nil
}

// Issues returns the list of issued tokens.
func (r *Request) Issues() []*Issue {
	var issues []*Issue
	for _, issue := range r.Metadata.Issues {
		issues = append(issues, &Issue{
			Issuer:    issue.Issuer,
			Receivers: issue.Receivers,
		})
	}
	return issues
}

// Transfers returns the list of transfers.
func (r *Request) Transfers() []*Transfer {
	var transfers []*Transfer
	for _, transfer := range r.Metadata.Transfers {
		transfers = append(transfers, &Transfer{
			Senders:      transfer.Senders,
			Receivers:    transfer.Receivers,
			ExtraSigners: transfer.ExtraSigners,
		})
	}
	return transfers
}

// AuditCheck performs the audit check of the request in addition to
// the checks of the token request itself via IsValid.
func (r *Request) AuditCheck() error {
	logger.Debugf("audit check request [%s] on tms [%s]", r.Anchor, r.TokenService.ID().String())
	if err := r.IsValid(); err != nil {
		return err
	}
	return r.TokenService.tms.AuditorCheck(
		r.Actions,
		r.Metadata,
		r.Anchor,
	)
}

// AuditRecord return the audit record of the request.
// The audit record contains: The anchor, the audit inputs and outputs
func (r *Request) AuditRecord() (*AuditRecord, error) {
	inputs, outputs, err := r.inputsAndOutputs(true, false)
	if err != nil {
		return nil, err
	}

	// load the tokens corresponding to the input token ids
	ids := inputs.IDs()
	toks, err := r.TokenService.Vault().NewQueryEngine().ListAuditTokens(ids...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed retrieving inputs for auditing")
	}
	if len(ids) != len(toks) {
		return nil, errors.Errorf("retrieved less inputs than those in the transaction [%d][%d]", len(ids), len(toks))
	}

	// populate type and quantity
	precision := r.TokenService.tms.PublicParamsManager().PublicParameters().Precision()
	for i := 0; i < len(ids); i++ {
		in := inputs.At(i)
		if toks[i] == nil {
			return nil, errors.Errorf("failed to audit inputs: nil input at [%d]th input", i)
		}
		in.Type = toks[i].Type
		q, err := token.ToQuantity(toks[i].Quantity, precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed converting quantity [%s]", toks[i].Quantity)
		}
		in.Quantity = q

		// retrieve the owner's audit info
		ownerAuditInfo, err := r.TokenService.tms.GetAuditInfo(toks[i].Owner.Raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for owner [%s]", toks[i].Owner)
		}
		in.OwnerAuditInfo = ownerAuditInfo
	}

	return &AuditRecord{
		Anchor:  r.Anchor,
		Inputs:  inputs,
		Outputs: outputs,
	}, nil
}

// ApplicationMetadata returns the application metadata corresponding to the given key
func (r *Request) ApplicationMetadata(k string) []byte {
	if len(r.Metadata.Application) == 0 {
		return nil
	}
	return r.Metadata.Application[k]
}

// SetApplicationMetadata sets application metadata in terms of key-value pairs.
// The Token-SDK does not control the format of the metadata.
func (r *Request) SetApplicationMetadata(k string, v []byte) {
	if r.Metadata == nil {
		r.Metadata = &driver.TokenRequestMetadata{}
	}
	if len(r.Metadata.Application) == 0 {
		r.Metadata.Application = map[string][]byte{}
	}
	r.Metadata.Application[k] = v
}

// FilterMetadataBy returns a new Request with the metadata filtered by the given enrollment IDs.
func (r *Request) FilterMetadataBy(eIDs ...string) (*Request, error) {
	meta := &Metadata{
		TMS:                  r.TokenService.tms,
		TokenRequestMetadata: r.Metadata,
	}
	filteredMeta, err := meta.FilterBy(eIDs[0])
	if err != nil {
		return nil, errors.WithMessagef(err, "failed filtering metadata by [%s]", eIDs[0])
	}
	return &Request{
		Anchor:       r.Anchor,
		Actions:      r.Actions,
		Metadata:     filteredMeta.TokenRequestMetadata,
		TokenService: r.TokenService,
	}, nil
}

// GetMetadata returns the metadata of the request.
func (r *Request) GetMetadata() (*Metadata, error) {
	if r.TokenService == nil {
		return nil, errors.New("can't get metadata: nil token service in request")
	}
	return &Metadata{
		TMS:                  r.TokenService.tms,
		TokenRequestMetadata: r.Metadata,
	}, nil
}

func (r *Request) parseInputIDs(inputs []*token.ID) ([]*token.ID, token.Quantity, string, error) {
	inputTokens, err := r.TokenService.Vault().NewQueryEngine().GetTokens(inputs...)
	if err != nil {
		return nil, nil, "", errors.WithMessagef(err, "failed querying tokens ids")
	}
	var typ string
	precision := r.TokenService.tms.PublicParamsManager().PublicParameters().Precision()
	sum := token.NewZeroQuantity(precision)
	for _, tok := range inputTokens {
		if len(typ) == 0 {
			typ = tok.Type
		}
		if typ != tok.Type {
			return nil, nil, "", errors.WithMessagef(err, "tokens must have the same type [%s]!=[%s]", typ, tok.Type)
		}
		q, err := token.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, nil, "", errors.WithMessagef(err, "failed unmarshalling token quantity [%s]", tok.Quantity)
		}
		sum = sum.Add(q)
	}

	return inputs, sum, typ, nil
}

func (r *Request) prepareTransfer(redeem bool, wallet *OwnerWallet, tokenType string, values []uint64, owners []view.Identity, transferOpts *TransferOptions) ([]*token.ID, []*token.Token, error) {
	for _, owner := range owners {
		if redeem {
			if !owner.IsNone() {
				return nil, nil, errors.Errorf("all recipients must be nil")
			}
		} else {
			if owner.IsNone() {
				return nil, nil, errors.Errorf("all recipients should be defined")
			}
		}
	}
	var tokenIDs []*token.ID
	var inputSum token.Quantity
	var err error
	// if inputs have been passed, parse and certify them, if needed
	if len(transferOpts.TokenIDs) != 0 {
		tokenIDs, inputSum, tokenType, err = r.parseInputIDs(transferOpts.TokenIDs)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed parsing passed input tokens")
		}
	}

	if tokenType == "" {
		return nil, nil, errors.Errorf("type is empty")
	}

	// Compute output tokens
	outputTokens, outputSum, err := r.genOutputs(values, owners, tokenType)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to generate outputs")
	}

	// Select input tokens, if not passed as opt
	if len(transferOpts.TokenIDs) == 0 {
		selector := transferOpts.Selector
		if selector == nil {
			// resort to default strategy
			selector, err = r.TokenService.SelectorManager().NewSelector(r.Anchor)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed getting default selector")
			}
		}
		tokenIDs, inputSum, err = selector.Select(wallet, outputSum.Decimal(), tokenType)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed selecting tokens")
		}
	}

	// Is there a rest?
	cmp := inputSum.Cmp(outputSum)
	switch cmp {
	case 1:
		diff := inputSum.Sub(outputSum)
		logger.Debugf("reassign rest [%s] to sender", diff.Decimal())

		pseudonym, err := wallet.GetRecipientIdentity()
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "failed getting recipient identity for the rest, wallet [%s]", wallet.ID())
		}

		outputTokens = append(outputTokens, &token.Token{
			Owner:    &token.Owner{Raw: pseudonym},
			Type:     tokenType,
			Quantity: diff.Hex(),
		})
	case -1:
		return nil, nil, errors.Errorf("the sum of the outputs is larger then the sum of the inputs [%s][%s]", inputSum.Decimal(), outputSum.Decimal())
	}

	if r.TokenService.PublicParametersManager().GraphHiding() {
		logger.Debugf("graph hiding enabled, request certification")
		// Check token certification
		cc, err := r.TokenService.CertificationClient()
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "cannot get certification client")
		}
		if err := cc.RequestCertification(tokenIDs...); err != nil {
			return nil, nil, errors.WithMessagef(err, "failed certifiying inputs")
		}
	}

	return tokenIDs, outputTokens, nil
}

func (r *Request) genOutputs(values []uint64, owners []view.Identity, tokenType string) ([]*token.Token, token.Quantity, error) {
	precision := r.TokenService.PublicParametersManager().Precision()
	maxTokenValue := r.TokenService.PublicParametersManager().MaxTokenValue()
	maxTokenValueQ, err := token.UInt64ToQuantity(maxTokenValue, precision)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to convert [%d] to quantity of precision [%d]", maxTokenValue, precision)
	}
	outputSum := token.NewZeroQuantity(precision)
	var outputTokens []*token.Token
	for i, value := range values {
		q, err := token.UInt64ToQuantity(value, precision)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert [%d] to quantity of precision [%d]", value, precision)
		}
		if q.Cmp(maxTokenValueQ) == 1 {
			return nil, nil, errors.Errorf("cannot create output with value [%s], max [%s]", q.Decimal(), maxTokenValueQ.Decimal())
		}
		outputSum = outputSum.Add(q)

		// single output is fine
		outputTokens = append(outputTokens, &token.Token{
			Owner:    &token.Owner{Raw: owners[i]},
			Type:     tokenType,
			Quantity: q.Hex(),
		})
	}
	return outputTokens, outputSum, nil
}

type requestSer struct {
	TxID     string
	Actions  []byte
	Metadata []byte
}
