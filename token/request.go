/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	// TransferMetadataPrefix is the prefix for the metadata of a transfer action
	TransferMetadataPrefix = meta.TransferMetadataPrefix
	// IssueMetadataPrefix is the prefix for the metadata of an issue action
	IssueMetadataPrefix = meta.IssueMetadataPrefix
	// PublicMetadataPrefix is the prefix for the metadata that will be published on the ledger without further validation
	PublicMetadataPrefix = meta.PublicMetadataPrefix
)

type Binder interface {
	Bind(ctx context.Context, longTerm Identity, ephemeral Identity) error
}

type (
	// TokensUpgradeChallenge is the challenge the issuer generates to make sure the client is not cheating
	TokensUpgradeChallenge = driver.TokensUpgradeChallenge
	// TokensUpgradeProof is the proof generated with the respect to a given challenge to prove the validity of the tokens to be upgrade
	TokensUpgradeProof = driver.TokensUpgradeProof
	// RequestAnchor models the anchor of a token request
	RequestAnchor = driver.TokenRequestAnchor
)

// RecipientData contains information about the identity of a token owner
type RecipientData = driver.RecipientData

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

// WithIssueMetadata sets issue action metadata
func WithIssueMetadata(key string, value []byte) IssueOption {
	return WithIssueAttribute(IssueMetadataPrefix+key, value)
}

// TransferOptions models the options that can be passed to the transfer command
type TransferOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
	// Selector is the custom token selector to use. If nil, the default will be used.
	Selector Selector
	// TokenIDs to transfer. If empty, the tokens will be selected.
	TokenIDs []*token.ID
	// RestRecipientIdentity TODO:
	RestRecipientIdentity *RecipientData
}

func CompileTransferOptions(opts ...TransferOption) (*TransferOptions, error) {
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

// WithPublicTransferMetadata adds any data to the public ledger that may be relevant to the application.
// It is also made available to the participants as part of the TransactionRecord.
// The transaction fails if the key already exists on the ledger. The value is not validated.
func WithPublicTransferMetadata(key string, value []byte) TransferOption {
	return WithTransferMetadata(PublicMetadataPrefix+key, value)
}

// WithPublicIssueMetadata adds any data to the public ledger that may be relevant to the application.
// It is also made available to the participants as part of the TransactionRecord.
// The transaction fails if the key already exists on the ledger. The value is not validated.
func WithPublicIssueMetadata(key string, value []byte) IssueOption {
	return WithIssueMetadata(PublicMetadataPrefix+key, value)
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

// WithRestRecipientIdentity sets the recipient data to be used to assign any rest left during a transfer operation
func WithRestRecipientIdentity(recipientData *RecipientData) TransferOption {
	return func(o *TransferOptions) error {
		o.RestRecipientIdentity = recipientData
		return nil
	}
}

// AuditRecord models the audit record returned by the audit command
// It contains the token request's anchor, inputs (with Type and Quantity), and outputs
type AuditRecord struct {
	// Anchor is used to bind the Actions to a given Transaction
	Anchor RequestAnchor
	// Inputs represent the input tokens of the transaction
	Inputs *InputStream
	// Outputs represent the output tokens of the transaction
	Outputs *OutputStream
	// Attributes are metadata which are stored on the public ledger as part of the transaction Actions.
	Attributes map[string][]byte
}

// Issue contains information about an issue operation.
// In particular, it carries the identities of the issuer and the receivers
type Issue struct {
	// Issuer is the issuer of the tokens
	Issuer Identity
	// Receivers is the list of identities of the receivers
	Receivers []Identity
	// ExtraSigners is the list of extra identities that must sign the token request to make it valid.
	// This field is to be used by the token drivers to list any additional identities that must
	// sign the token request.
	ExtraSigners []Identity
}

// Transfer contains information about a transfer operation.
// In particular, it carries the identities of the senders and the receivers
type Transfer struct {
	// Senders is the list of identities of the senders
	Senders []Identity
	// Receivers is the list of identities of the receivers
	Receivers []Identity
	// ExtraSigners is the list of extra identities that must sign the token request to make it valid.
	// This field is to be used by the token drivers to list any additional identities that must
	// sign the token request.
	ExtraSigners []Identity
	// Issuer
	Issuer Identity
}

// Request aggregates token operations that must be performed atomically.
// Operations are represented in a backend agnostic way but driver specific.
type Request struct {
	// Anchor is used to bind the Actions to a given Transaction
	Anchor driver.TokenRequestAnchor
	// Actions contains the token operations.
	Actions *driver.TokenRequest
	// Metadata contains the actions' metadata used to unscramble the content of the actions, if the
	// underlying token driver requires that
	Metadata *driver.TokenRequestMetadata
	// TokenService this request refers to
	TokenService *ManagementService `json:"-"`
}

// NewRequest creates a new empty request for the given token service and anchor
func NewRequest(tokenService *ManagementService, anchor RequestAnchor) *Request {
	return &Request{
		Anchor:       anchor,
		Actions:      &driver.TokenRequest{},
		Metadata:     &driver.TokenRequestMetadata{},
		TokenService: tokenService,
	}
}

// NewRequestFromBytes creates a new request from the given anchor, and whose actions and metadata
// are unmarshalled from the given bytes
func NewRequestFromBytes(tokenService *ManagementService, anchor RequestAnchor, actions []byte, trmRaw []byte) (*Request, error) {
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

// NewFullRequestFromBytes creates a new request from the given byte representation
func NewFullRequestFromBytes(tokenService *ManagementService, tr []byte) (*Request, error) {
	request := NewRequest(tokenService, "")
	if err := request.FromBytes(tr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal request")
	}
	return request, nil
}

// ID returns the anchor of the request
func (r *Request) ID() RequestAnchor {
	return r.Anchor
}

// Issue appends an issue action to the request. The action will be prepared using the provided issuer wallet.
// The action issues to the receiver a token of the passed type and quantity.
// Additional options can be passed to customize the action.
func (r *Request) Issue(ctx context.Context, wallet *IssuerWallet, receiver Identity, typ token.Type, q uint64, opts ...IssueOption) (*IssueAction, error) {
	logger.DebugfContext(ctx, "Start issue")
	logger.DebugfContext(ctx, "Done issue")
	if wallet == nil {
		return nil, errors.Errorf("wallet is nil")
	}
	if typ == "" {
		return nil, errors.Errorf("type is empty")
	}
	if q == 0 {
		return nil, errors.Errorf("q is zero")
	}
	maxTokenValue := r.TokenService.PublicParametersManager().PublicParameters().MaxTokenValue()
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
	action, metaRaw, err := r.TokenService.tms.IssueService().Issue(
		ctx,
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
	actionRaw, err := action.Serialize()
	if err != nil {
		return nil, err
	}
	r.Actions.Issues = append(r.Actions.Issues, actionRaw)
	r.Metadata.Issues = append(r.Metadata.Issues, metaRaw)

	return &IssueAction{a: action}, nil
}

// Transfer appends a transfer action to the request. The action will be prepared using the provided owner wallet.
// The action transfers tokens of the passed types to the receivers for the passed quantities.
// In other words, owners[0] will receives values[0], and so on.
// Additional options can be passed to customize the action.
func (r *Request) Transfer(ctx context.Context, wallet *OwnerWallet, typ token.Type, values []uint64, owners []Identity, opts ...TransferOption) (*TransferAction, error) {
	for _, v := range values {
		if v == 0 {
			return nil, errors.Errorf("value is zero")
		}
	}
	opt, err := CompileTransferOptions(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed compiling options [%v]", opts)
	}
	tokenIDs, outputTokens, err := r.prepareTransfer(ctx, false, wallet, typ, values, owners, opt)
	if err != nil {
		return nil, errors.Wrap(err, "failed preparing transfer")
	}

	r.TokenService.logger.DebugfContext(ctx, "Prepare Transfer Action [id:%s,ins:%d,outs:%d,attr:%d]", r.Anchor, len(tokenIDs), len(outputTokens), len(opt.Attributes))

	ts := r.TokenService.tms.TransferService()

	// Compute transfer
	transfer, transferMetadata, err := ts.Transfer(
		ctx,
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
	if r.TokenService.logger.IsEnabledFor(zapcore.DebugLevel) {
		// double check
		if err := ts.VerifyTransfer(ctx, transfer, transferMetadata.Outputs); err != nil {
			return nil, errors.Wrap(err, "failed checking generated proof")
		}
	}

	// Append
	raw, err := transfer.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing transfer action")
	}
	r.Actions.Transfers = append(r.Actions.Transfers, raw)
	r.Metadata.Transfers = append(r.Metadata.Transfers, transferMetadata)

	return &TransferAction{TransferAction: transfer}, nil
}

// Redeem appends a redeem action to the request. The action will be prepared using the provided owner wallet.
// The action redeems tokens of the passed type for a total amount matching the passed value.
// Additional options can be passed to customize the action.
func (r *Request) Redeem(ctx context.Context, wallet *OwnerWallet, typ token.Type, value uint64, opts ...TransferOption) (*TransferAction, error) {
	opt, err := CompileTransferOptions(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed compiling options [%v]", opts)
	}
	tokenIDs, outputTokens, err := r.prepareTransfer(ctx, true, wallet, typ, []uint64{value}, []Identity{nil}, opt)
	if err != nil {
		return nil, errors.Wrap(err, "failed preparing transfer")
	}

	r.TokenService.logger.DebugfContext(ctx, "Prepare Redeem Action [ins:%d,outs:%d]", len(tokenIDs), len(outputTokens))

	ts := r.TokenService.tms.TransferService()

	// Compute redeem, it is a transfer with owner set to nil
	transfer, transferMetadata, err := ts.Transfer(
		ctx,
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

	if r.TokenService.logger.IsEnabledFor(zapcore.DebugLevel) {
		// double check
		if err := ts.VerifyTransfer(ctx, transfer, transferMetadata.Outputs); err != nil {
			return nil, errors.Wrap(err, "failed checking generated proof")
		}
	}

	// Append
	raw, err := transfer.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing transfer action")
	}

	r.Actions.Transfers = append(r.Actions.Transfers, raw)
	r.Metadata.Transfers = append(r.Metadata.Transfers, transferMetadata)

	return &TransferAction{transfer}, nil
}

// Upgrade performs an upgrade operation of the passed ledger tokens.
// A proof and its challenge will be used to verify that the request of upgrade is legit.
// If the proof verifies then the passed wallet will be used to issue a new amount of tokens
// matching those whose upgrade has been requested.
func (r *Request) Upgrade(
	ctx context.Context,
	wallet *IssuerWallet,
	receiver Identity,
	challenge TokensUpgradeChallenge,
	tokens []token.LedgerToken,
	proof TokensUpgradeProof,
	opts ...IssueOption,
) (*IssueAction, error) {
	if wallet == nil {
		return nil, errors.Errorf("wallet is nil")
	}
	if len(tokens) == 0 {
		return nil, errors.Errorf("tokens is empty")
	}

	opt, err := compileIssueOptions(opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed compiling options [%v]", opts)
	}

	// Compute Issue
	action, meta, err := r.TokenService.tms.IssueService().Issue(
		ctx,
		nil,
		"",
		nil,
		[][]byte{receiver},
		&driver.IssueOptions{
			Attributes: opt.Attributes,
			TokensUpgradeRequest: &driver.TokenUpgradeRequest{
				Challenge: challenge,
				Tokens:    tokens,
				Proof:     proof,
			},
			Wallet: wallet.w,
		},
	)
	if err != nil {
		return nil, err
	}

	// Append
	raw, err := action.Serialize()
	if err != nil {
		return nil, err
	}
	r.Actions.Issues = append(r.Actions.Issues, raw)
	r.Metadata.Issues = append(r.Metadata.Issues, meta)

	return &IssueAction{a: action}, nil
}

// Outputs returns the sequence of outputs of the request supporting sequential and parallel aggregate operations.
func (r *Request) Outputs() (*OutputStream, error) {
	return r.outputs(false)
}

func (r *Request) outputs(failOnMissing bool) (*OutputStream, error) {
	tms := r.TokenService.tms
	pp := tms.PublicParamsManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public paramenters not set")
	}

	meta, err := r.GetMetadata()
	if err != nil {
		return nil, err
	}
	var outputs []*Output
	counter := uint64(0)
	is := tms.IssueService()
	for i, issue := range r.Actions.Issues {
		// deserialize action
		issueAction, err := is.DeserializeIssueAction(issue)
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

		extractedOutputs, newCounter, err := r.extractIssueOutputs(i, counter, issueAction, issueMeta, failOnMissing, false)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter = newCounter
	}

	ts := tms.TransferService()
	for i, transfer := range r.Actions.Transfers {
		// deserialize action
		transferAction, err := ts.DeserializeTransferAction(transfer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing transfer action [%d]", i)
		}
		// get metadata for action
		transferMeta, err := meta.Transfer(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting transfer metadata [%d]", i)
		}
		if err := transferMeta.Match(&TransferAction{transferAction}); err != nil {
			return nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]", i)
		}
		if len(transferAction.GetOutputs()) != len(transferMeta.Outputs) {
			return nil, errors.Errorf("failed matching transfer action with its metadata [%d]: invalid metadata", i)
		}
		extractedOutputs, newCounter, err := r.extractTransferOutputs(i, counter, transferAction, transferMeta, failOnMissing, false)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter = newCounter
	}
	return NewOutputStream(outputs, tms.PublicParamsManager().PublicParameters().Precision()), nil
}

func (r *Request) extractIssueOutputs(i int, counter uint64, issueAction driver.IssueAction, issueMeta *IssueMetadata, failOnMissing, noOutputForRecipient bool) ([]*Output, uint64, error) {
	if len(issueAction.GetOutputs()) != len(issueMeta.Outputs) {
		return nil, 0, errors.Errorf("failed matching issue action with its metadata [%d]: invalid metadata, the number of outputs does not match", i)
	}
	// extract outputs for this action
	tms := r.TokenService.tms
	pp := tms.PublicParamsManager().PublicParameters()
	if pp == nil {
		return nil, 0, errors.Errorf("public paramenters not set")
	}
	precision := pp.Precision()
	var outputs []*Output
	for j, output := range issueAction.GetOutputs() {
		if output == nil {
			return nil, 0, errors.Errorf("%d^th output in issue action [%d] is nil", j, i)
		}

		raw, err := output.Serialize()
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed deserializing issue action output [%d,%d]", i, j)
		}

		// is the j-th meta present? It might have been filtered out
		if issueMeta.IsOutputAbsent(j) {
			r.TokenService.logger.Debugf("Issue Action Output [%d,%d] is absent", i, j)
			if failOnMissing {
				return nil, 0, errors.Errorf("missing token info for output [%d,%d]", i, j)
			}
			// // check the recipients anyway
			// recipients, err := tms.TokensService().Recipients(raw)
			// if err != nil {
			// 	return nil, 0, errors.Wrapf(err, "failed getting recipients [%d,%d]", i, j)
			// }
			// for k, recipient := range recipients {
			// 	metaRecipient := issueMeta.Outputs[j].RecipientAt(k)
			// 	if metaRecipient == nil {
			// 		return nil, 0, errors.Errorf("missing recipient metadata for output [%d,%d]", i, j)
			// 	}
			// 	if !recipient.Equal(metaRecipient.Identity) {
			// 		return nil, 0, errors.Errorf("invalid recipient [%d,%d] [%s:%s]", i, j, recipient, metaRecipient.Identity)
			// 	}
			// }
			counter++
			continue
		}

		// is the j-th meta present? Yes
		tok, issuer, recipients, format, err := tms.TokensService().Deobfuscate(raw, issueMeta.Outputs[j].OutputMetadata)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed getting issue action output in the clear [%d,%d]", i, j)
		}
		if !issuer.Equal(issueAction.GetIssuer()) {
			return nil, 0, errors.Errorf("invalid issuer [%d,%d]", i, j)
		}
		if len(recipients) == 0 {
			return nil, 0, errors.Errorf("missing recipients [%d,%d]", i, j)
		}
		q, err := token.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed getting quantity [%d,%d]", i, j)
		}
		if noOutputForRecipient {
			outputs = append(outputs, &Output{
				Token:                *tok,
				ActionIndex:          i,
				Index:                counter,
				Owner:                tok.Owner,
				Type:                 tok.Type,
				Quantity:             q,
				Issuer:               issuer,
				LedgerOutput:         raw,
				LedgerOutputFormat:   format,
				LedgerOutputMetadata: issueMeta.Outputs[j].OutputMetadata,
			})
		} else {
			for k, recipient := range recipients {
				metaRecipient := issueMeta.Outputs[j].RecipientAt(k)
				if metaRecipient == nil {
					return nil, 0, errors.Errorf("missing recipient metadata for output [%d,%d]", i, j)
				}
				if !recipient.Equal(metaRecipient.Identity) {
					return nil, 0, errors.Errorf("invalid recipient [%d,%d] [%s:%s]", i, j, recipient, metaRecipient.Identity)
				}
				eID, rID, err := tms.WalletService().GetEIDAndRH(recipient, metaRecipient.AuditInfo)
				if err != nil {
					return nil, 0, errors.Wrapf(err, "failed getting enrollment id and revocation handle [%d,%d]", i, j)
				}

				outputs = append(outputs, &Output{
					Token:                *tok,
					ActionIndex:          i,
					Index:                counter,
					Owner:                recipient,
					OwnerAuditInfo:       metaRecipient.AuditInfo,
					EnrollmentID:         eID,
					RevocationHandler:    rID,
					Type:                 tok.Type,
					Quantity:             q,
					Issuer:               issuer,
					LedgerOutput:         raw,
					LedgerOutputFormat:   format,
					LedgerOutputMetadata: issueMeta.Outputs[j].OutputMetadata,
				})
			}
		}
		counter++
	}
	return outputs, counter, nil
}

func (r *Request) extractTransferOutputs(i int, counter uint64, transferAction driver.TransferAction, transferMeta *TransferMetadata, failOnMissing, noOutputForRecipient bool) ([]*Output, uint64, error) {
	tms := r.TokenService.tms
	if tms.PublicParamsManager() == nil || tms.PublicParamsManager().PublicParameters() == nil {
		return nil, 0, errors.New("can't get inputs: invalid token service in request")
	}
	precision := tms.PublicParamsManager().PublicParameters().Precision()
	var outputs []*Output
	recipientCounter := 0
	for j, output := range transferAction.GetOutputs() {
		if output == nil {
			return nil, 0, errors.Errorf("%d^th output in transfer action [%d] is nil", j, i)
		}
		ledgerOutput, err := output.Serialize()
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed deserializing transfer action output [%d,%d]", i, j)
		}
		// is the j-th meta present? It might have been filtered out
		if transferMeta.IsOutputAbsent(j) || len(transferMeta.Outputs[j].OutputMetadata) == 0 {
			r.TokenService.logger.Debugf("Transfer Action Output [%d,%d] is absent", i, j)
			if failOnMissing {
				return nil, 0, errors.Errorf("missing token info for output [%d,%d]", i, j)
			}
			// check the recipients anyway
			// recipients, err := tms.TokensService().Recipients(ledgerOutput)
			// if err != nil {
			// 	return nil, 0, errors.Wrapf(err, "failed getting recipients [%d,%d]", i, j)
			// }
			// for k, recipient := range recipients {
			// 	metaRecipient := transferMeta.Outputs[j].RecipientAt(k)
			// 	if metaRecipient == nil {
			// 		return nil, 0, errors.Errorf("missing recipient metadata for output [%d,%d]", i, j)
			// 	}
			// 	if !recipient.Equal(metaRecipient.Identity) {
			// 		return nil, 0, errors.Errorf("invalid recipient [%d,%d] [%s:%s]", i, j, recipient, metaRecipient.Identity)
			// 	}
			// }
			counter++
			continue
		}
		// is the j-th meta present? Yes
		tok, issuer, recipients, ledgerOutputFormat, err := tms.TokensService().Deobfuscate(ledgerOutput, transferMeta.Outputs[j].OutputMetadata)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed getting transfer action output in the clear [%d,%d]", i, j)
		}
		if len(recipients) == 0 {
			// Add an empty recipient
			recipients = append(recipients, Identity{})
		}

		q, err := token.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed getting quantity [%d,%d]", i, j)
		}
		if noOutputForRecipient {
			outputs = append(outputs, &Output{
				Token:                *tok,
				ActionIndex:          i,
				Index:                counter,
				Owner:                tok.Owner,
				OwnerAuditInfo:       transferMeta.Outputs[j].OutputAuditInfo,
				EnrollmentID:         "", // not available here
				RevocationHandler:    "", // not available here
				Type:                 tok.Type,
				Quantity:             q,
				LedgerOutput:         ledgerOutput,
				LedgerOutputFormat:   ledgerOutputFormat,
				LedgerOutputMetadata: transferMeta.Outputs[j].OutputMetadata,
				Issuer:               issuer,
			})
			for k, recipient := range recipients {
				metaRecipient := transferMeta.Outputs[j].RecipientAt(k)
				if metaRecipient == nil {
					return nil, 0, errors.Errorf("missing recipient metadata for output [%d,%d]", i, j)
				}
				if !recipient.Equal(metaRecipient.Identity) {
					return nil, 0, errors.Errorf("invalid recipient [%d,%d] [%s:%s]", i, j, recipient, metaRecipient.Identity)
				}
			}
		} else {
			for k, recipient := range recipients {
				metaRecipient := transferMeta.Outputs[j].RecipientAt(k)
				if metaRecipient == nil {
					return nil, 0, errors.Errorf("missing recipient metadata for output [%d,%d]", i, j)
				}
				if !recipient.Equal(metaRecipient.Identity) {
					return nil, 0, errors.Errorf("invalid recipient [%d,%d] [%s:%s]", i, j, recipient, metaRecipient.Identity)
				}
				var eID string
				var rID string
				var receiverAuditInfo []byte
				var targetLedgerOutput []byte
				if len(tok.Owner) != 0 {
					receiverAuditInfo = metaRecipient.AuditInfo
					eID, rID, err = tms.WalletService().GetEIDAndRH(recipient, receiverAuditInfo)
					if err != nil {
						return nil, 0, errors.Wrapf(err, "failed getting enrollment id and revocation handle [%d,%d]", i, recipientCounter)
					}
					targetLedgerOutput = ledgerOutput
				}
				r.TokenService.logger.Debugf("Transfer Action Output [%d,%d][%s:%d] is present, extract [%s]", i, j, r.Anchor, counter, Hashable(ledgerOutput))
				outputs = append(outputs, &Output{
					Token:                *tok,
					ActionIndex:          i,
					Index:                counter,
					Owner:                recipient,
					OwnerAuditInfo:       receiverAuditInfo,
					EnrollmentID:         eID,
					RevocationHandler:    rID,
					Type:                 tok.Type,
					Quantity:             q,
					LedgerOutput:         targetLedgerOutput,
					LedgerOutputFormat:   ledgerOutputFormat,
					LedgerOutputMetadata: transferMeta.Outputs[j].OutputMetadata,
					Issuer:               issuer,
				})
				recipientCounter++
			}
		}
		counter++
	}
	return outputs, counter, nil
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
	ts := tms.TransferService()
	for i, transfer := range r.Actions.Transfers {
		// deserialize action
		transferAction, err := ts.DeserializeTransferAction(transfer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing transfer action [%d]", i)
		}
		// get metadata for action
		transferMeta, err := meta.Transfer(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting transfer metadata [%d]", i)
		}
		if err := transferMeta.Match(&TransferAction{transferAction}); err != nil {
			return nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]", i)
		}

		// we might not have TokenIDs if they have been filtered
		if len(transferMeta.Inputs) == 0 && failOnMissing {
			return nil, errors.Errorf("missing token ids for transfer [%d]", i)
		}

		extractedInputs, err := r.extractTransferInputs(i, transferMeta, failOnMissing)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, extractedInputs...)
	}
	return NewInputStream(r.TokenService.Vault().NewQueryEngine(), inputs, tms.PublicParamsManager().PublicParameters().Precision()), nil
}

func (r *Request) extractIssueInputs(actionIndex int, metadata *IssueMetadata) ([]*Input, error) {
	var inputs []*Input
	for _, input := range metadata.Inputs {
		inputs = append(inputs, &Input{
			ActionIndex: actionIndex,
			Id:          input.TokenID,
		})
	}
	return inputs, nil
}

func (r *Request) extractTransferInputs(actionIndex int, metadata *TransferMetadata, failOnMissing bool) ([]*Input, error) {
	// Iterate over the metadata.SenderAuditInfos because we know that there will be at least one
	// sender, but it might be that there are not token IDs due to filtering.
	tms := r.TokenService.tms
	var inputs []*Input
	for j, input := range metadata.Inputs {
		// The recipient might be missing because it has been filtered out. Skip in this case
		if metadata.IsInputAbsent(j) {
			if failOnMissing {
				return nil, errors.Errorf("missing receiver for transfer [%d,%d]", actionIndex, j)
			}
			continue
		}

		for _, sender := range input.Senders {
			eID, rID, err := tms.WalletService().GetEIDAndRH(sender.Identity, sender.AuditInfo)
			if err != nil {
				return nil, errors.Wrapf(err, "failed getting enrollment id and revocation handle [%d,%d]", actionIndex, j)
			}

			inputs = append(inputs, &Input{
				ActionIndex:       actionIndex,
				Id:                metadata.TokenIDAt(j),
				Owner:             sender.Identity,
				OwnerAuditInfo:    sender.AuditInfo,
				EnrollmentID:      eID,
				RevocationHandler: rID,
			})
		}
	}
	return inputs, nil
}

func (r *Request) InputsAndOutputs(ctx context.Context) (*InputStream, *OutputStream, map[string][]byte, error) {
	return r.inputsAndOutputs(ctx, false, false, false)
}

func (r *Request) InputsAndOutputsNoRecipients(ctx context.Context) (*InputStream, *OutputStream, error) {
	is, os, _, err := r.inputsAndOutputs(ctx, false, false, true)
	return is, os, err
}

func (r *Request) inputsAndOutputs(ctx context.Context, failOnMissing, verifyActions, noOutputForRecipient bool) (*InputStream, *OutputStream, map[string][]byte, error) {
	tms := r.TokenService.tms
	if tms.PublicParamsManager() == nil || tms.PublicParamsManager().PublicParameters() == nil {
		return nil, nil, nil, errors.New("can't get inputs: invalid token service in request")
	}
	meta, err := r.GetMetadata()
	if err != nil {
		return nil, nil, nil, err
	}
	var inputs []*Input
	var outputs []*Output
	attributes := map[string][]byte{}
	counter := uint64(0)

	issueService := tms.IssueService()
	for i, issue := range r.Actions.Issues {
		// deserialize action
		issueAction, err := issueService.DeserializeIssueAction(issue)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed deserializing issue action [%d]", i)
		}
		for k, v := range issueAction.GetMetadata() {
			attributes[k] = v
		}

		// get metadata for action
		issueMeta, err := meta.Issue(i)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed getting issue metadata [%d]", i)
		}
		if err := issueMeta.Match(&IssueAction{a: issueAction}); err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed matching issue action with its metadata [%d]", i)
		}

		if verifyActions {
			if err := issueService.VerifyIssue(issueAction, issueMeta.Outputs); err != nil {
				return nil, nil, nil, errors.WithMessagef(err, "failed verifying issue action")
			}
		}

		extractedInputs, err := r.extractIssueInputs(i, issueMeta)
		if err != nil {
			return nil, nil, nil, err
		}
		inputs = append(inputs, extractedInputs...)

		extractedOutputs, newCounter, err := r.extractIssueOutputs(i, counter, issueAction, issueMeta, failOnMissing, noOutputForRecipient)
		if err != nil {
			return nil, nil, nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter = newCounter
	}

	ts := tms.TransferService()
	for i, transfer := range r.Actions.Transfers {
		// deserialize action
		transferAction, err := ts.DeserializeTransferAction(transfer)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed deserializing transfer action [%d]", i)
		}
		for k, v := range transferAction.GetMetadata() {
			attributes[k] = v
		}
		// get metadata for action
		transferMeta, err := meta.Transfer(i)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed getting transfer metadata [%d]", i)
		}
		if err := transferMeta.Match(&TransferAction{transferAction}); err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed matching transfer action with its metadata [%d]", i)
		}
		if verifyActions {
			if err := ts.VerifyTransfer(ctx, transferAction, transferMeta.Outputs); err != nil {
				return nil, nil, nil, errors.WithMessagef(err, "failed verifying transfer action")
			}
		}

		// we might not have TokenIDs if they have been filtered
		if len(transferMeta.Inputs) == 0 && failOnMissing {
			return nil, nil, nil, errors.Errorf("missing token ids for transfer [%d]", i)
		}

		extractedInputs, err := r.extractTransferInputs(i, transferMeta, failOnMissing)
		if err != nil {
			return nil, nil, nil, err
		}
		inputs = append(inputs, extractedInputs...)

		extractedOutputs, newCounter, err := r.extractTransferOutputs(i, counter, transferAction, transferMeta, failOnMissing, noOutputForRecipient)
		if err != nil {
			return nil, nil, nil, err
		}
		outputs = append(outputs, extractedOutputs...)
		counter = newCounter
	}

	precision := tms.PublicParamsManager().PublicParameters().Precision()
	inputStream := NewInputStream(r.TokenService.Vault().NewQueryEngine(), inputs, precision)
	os := NewOutputStream(outputs, precision)
	return inputStream, os, attributes, nil
}

// IsValid checks that the request is valid.
func (r *Request) IsValid(ctx context.Context) error {
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
	if _, _, _, err := r.inputsAndOutputs(ctx, false, true, false); err != nil {
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
	return r.Actions.MarshalToMessageToSign([]byte(r.Anchor))
}

// MarshalToSign marshals the request to a message suitable for signing.
func (r *Request) MarshalToSign() ([]byte, error) {
	if r.Actions == nil {
		return nil, errors.Errorf("failed to marshal request in tx [%s] for signing", r.Anchor)
	}
	return r.Actions.MarshalToMessageToSign([]byte(r.Anchor))
}

// RequestToBytes marshals the request's actions to bytes.
func (r *Request) RequestToBytes() ([]byte, error) {
	if r.Actions == nil {
		return nil, errors.Errorf("failed to marshal request in tx [%s]", r.Anchor)
	}
	return r.Actions.Bytes()
}

// Bytes marshals the request to bytes.
// It includes: Anchor (or ID), actions, and metadata.
func (r *Request) Bytes() ([]byte, error) {
	requestProto, err := r.Actions.ToProtos()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal request in tx [%s]", r.Anchor)
	}
	metadataProto, err := r.Metadata.ToProtos()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal metadata in tx [%s]", r.Anchor)
	}
	requestWithMetadata := &request.TokenRequestWithMetadata{
		Version:  driver.ProtocolV1,
		Anchor:   string(r.Anchor),
		Request:  requestProto,
		Metadata: metadataProto,
	}
	return proto.Marshal(requestWithMetadata)
}

// FromBytes unmarshalls the request from bytes overriding the content of the current request.
func (r *Request) FromBytes(raw []byte) error {
	requestWithMetadata := &request.TokenRequestWithMetadata{}
	if err := proto.Unmarshal(raw, requestWithMetadata); err != nil {
		return errors.Wrapf(err, "failed unmarshaling request")
	}
	// assert version
	if requestWithMetadata.Version != driver.ProtocolV1 {
		return errors.Errorf("invalid token request with metadata version, expected [%d], got [%d]", driver.ProtocolV1, requestWithMetadata.Version)
	}

	r.Anchor = RequestAnchor(requestWithMetadata.Anchor)
	if requestWithMetadata.Request != nil {
		if err := r.Actions.FromProtos(requestWithMetadata.Request); err != nil {
			return errors.Wrapf(err, "failed unmarshalling actions")
		}
	}
	if requestWithMetadata.Metadata != nil {
		if err := r.Metadata.FromProtos(requestWithMetadata.Metadata); err != nil {
			return errors.Wrapf(err, "failed unmarshalling metadata")
		}
	}
	return nil
}

// AddAuditorSignature adds an auditor signature to the request.
func (r *Request) AddAuditorSignature(identity Identity, sigma []byte) {
	r.Actions.AuditorSignatures = append(r.Actions.AuditorSignatures, &driver.AuditorSignature{
		Identity:  identity,
		Signature: sigma,
	})
}

func (r *Request) SetSignatures(sigmas map[string][]byte) bool {
	signers := append(r.IssueSigners(), r.TransferSigners()...)
	signatures := make([][]byte, len(signers))
	all := true
	for i, signer := range signers {
		if sigma, ok := sigmas[signer.UniqueID()]; ok {
			signatures[i] = sigma
			r.TokenService.logger.Debugf("signature [%d] for signer [%s] is [%s]", i, signer, hash.Hashable(sigma))
		} else {
			all = false
			r.TokenService.logger.Debugf("signature [%d] for signer [%s] not found", i, signer)
		}
	}
	r.Actions.Signatures = signatures
	return all
}

func (r *Request) TransferSigners() []Identity {
	signers := make([]Identity, 0)
	for _, transfer := range r.Transfers() {
		signers = append(signers, transfer.Senders...)
		if transfer.Issuer != nil { // add also the identity of the issuer, if specified
			signers = append(signers, transfer.Issuer)
		}
		signers = append(signers, transfer.ExtraSigners...)
	}
	return signers
}

func (r *Request) IssueSigners() []Identity {
	signers := make([]Identity, 0)
	for _, issue := range r.Issues() {
		signers = append(signers, issue.Issuer)
		signers = append(signers, issue.ExtraSigners...)
	}
	return signers
}

// SetTokenService sets the token service.
func (r *Request) SetTokenService(service *ManagementService) {
	r.TokenService = service
}

// BindTo binds transfers' senders and receivers, that are senders, that are not me to the passed identity
func (r *Request) BindTo(ctx context.Context, binder Binder, identity Identity) error {
	for i := range r.Actions.Transfers {
		// senders
		for _, input := range r.Metadata.Transfers[i].Inputs {
			for _, sender := range input.Senders {
				senderIdentity := sender.Identity
				if w := r.TokenService.WalletManager().Wallet(ctx, senderIdentity); w != nil {
					// this is me, skip
					continue
				}
				r.TokenService.logger.DebugfContext(ctx, "bind sender [%s] to [%s]", senderIdentity, identity)
				if err := binder.Bind(ctx, identity, senderIdentity); err != nil {
					return errors.Wrap(err, "failed binding sender identities")
				}
			}
		}

		// extra signers
		for _, eid := range r.Metadata.Transfers[i].ExtraSigners {
			if w := r.TokenService.WalletManager().Wallet(ctx, eid); w != nil {
				// this is me, skip
				continue
			}
			r.TokenService.logger.DebugfContext(ctx, "bind extra signer [%s] to [%s]", eid, identity)
			if err := binder.Bind(ctx, identity, eid); err != nil {
				return errors.Wrap(err, "failed binding sender identities")
			}
		}

		// receivers
		for _, output := range r.Metadata.Transfers[i].Outputs {
			for _, receiver := range output.Receivers {
				receiverIdentity := receiver.Identity
				if w := r.TokenService.WalletManager().Wallet(ctx, receiverIdentity); w != nil {
					// this is me, skip
					continue
				}

				r.TokenService.logger.DebugfContext(ctx, "bind receiver as sender [%s] to [%s]", receiverIdentity, identity)
				if err := binder.Bind(ctx, identity, receiverIdentity); err != nil {
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
			Issuer:       issue.Issuer.Identity,
			Receivers:    issue.Receivers(),
			ExtraSigners: issue.ExtraSigners,
		})
	}
	return issues
}

// Transfers returns the list of transfers.
func (r *Request) Transfers() []*Transfer {
	var transfers []*Transfer
	for _, transfer := range r.Metadata.Transfers {
		transfers = append(transfers, &Transfer{
			Senders:      transfer.Senders(),
			Receivers:    transfer.Receivers(),
			ExtraSigners: transfer.ExtraSigners,
			Issuer:       transfer.Issuer,
		})
	}
	return transfers
}

// AuditCheck performs the audit check of the request in addition to
// the checks of the token request itself via IsValid.
func (r *Request) AuditCheck(ctx context.Context) error {
	r.TokenService.logger.DebugfContext(ctx, "audit check request [%s] on tms [%s]", r.Anchor, r.TokenService.ID())
	if err := r.IsValid(ctx); err != nil {
		return err
	}
	return r.TokenService.tms.AuditorService().AuditorCheck(
		ctx,
		r.Actions,
		r.Metadata,
		r.Anchor,
	)
}

// AuditRecord return the audit record of the request.
// The audit record contains: The anchor, the audit inputs and outputs
func (r *Request) AuditRecord(ctx context.Context) (*AuditRecord, error) {
	inputs, outputs, attr, err := r.inputsAndOutputs(ctx, true, false, false)
	if err != nil {
		return nil, err
	}

	// load the tokens corresponding to the input token ids
	ids := inputs.IDs()
	toks, err := r.TokenService.Vault().NewQueryEngine().ListAuditTokens(ctx, ids...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed retrieving inputs for auditing")
	}
	if len(ids) != len(toks) {
		return nil, errors.Errorf("retrieved less inputs than those in the transaction [%d][%d]", len(ids), len(toks))
	}

	// populate type and quantity
	pp := r.TokenService.tms.PublicParamsManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public paramenters not set")
	}
	precision := pp.Precision()
	for i := range ids {
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
		ownerAuditInfo, err := r.TokenService.tms.WalletService().GetAuditInfo(ctx, toks[i].Owner)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for owner [%s]", toks[i].Owner)
		}
		in.OwnerAuditInfo = ownerAuditInfo
	}

	return &AuditRecord{
		Anchor:     r.Anchor,
		Inputs:     inputs,
		Outputs:    outputs,
		Attributes: attr,
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
		TokenService:         r.TokenService.tms.TokensService(),
		WalletService:        r.TokenService.tms.WalletService(),
		TokenRequestMetadata: r.Metadata,
		Logger:               r.TokenService.logger,
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
	return &Metadata{
		TokenService:         r.TokenService.tms.TokensService(),
		WalletService:        r.TokenService.tms.WalletService(),
		TokenRequestMetadata: r.Metadata,
		Logger:               r.TokenService.logger,
	}, nil
}

func (r *Request) AllApplicationMetadata() map[string][]byte { return r.Metadata.Application }

func (r *Request) PublicParamsHash() PPHash {
	return r.TokenService.PublicParametersManager().PublicParamsHash()
}

func (r *Request) String() string { return string(r.Anchor) }

func (r *Request) parseInputIDs(ctx context.Context, inputs []*token.ID) ([]*token.ID, token.Quantity, token.Type, error) {
	inputTokens, err := r.TokenService.Vault().NewQueryEngine().GetTokens(ctx, inputs...)
	if err != nil {
		return nil, nil, "", errors.WithMessagef(err, "failed querying tokens ids")
	}
	var typ token.Type
	pp := r.TokenService.tms.PublicParamsManager().PublicParameters()
	if pp == nil {
		return nil, nil, "", errors.Errorf("public paramenters not set")
	}
	precision := pp.Precision()
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

func (r *Request) prepareTransfer(ctx context.Context, redeem bool, wallet *OwnerWallet, tokenType token.Type, values []uint64, owners []Identity, transferOpts *TransferOptions) ([]*token.ID, []*token.Token, error) {
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

	transferOpts.TokenIDs = r.cleanupInputIDs(transferOpts.TokenIDs)

	// if inputs have been passed, parse and certify them, if needed
	if len(transferOpts.TokenIDs) != 0 {
		tokenIDs, inputSum, tokenType, err = r.parseInputIDs(ctx, transferOpts.TokenIDs)
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
			sm, err := r.TokenService.SelectorManager()
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get selector manager")
			}
			selector, err = sm.NewSelector(string(r.Anchor))
			defer utils.IgnoreErrorWithOneArg(sm.Close, string(r.Anchor))
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed getting default selector")
			}
		}
		tokenIDs, inputSum, err = selector.Select(ctx, wallet, outputSum.Decimal(), tokenType)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed selecting tokens")
		}
	}

	// Is there a rest?
	cmp := inputSum.Cmp(outputSum)
	switch cmp {
	case 1:
		diff := inputSum.Sub(outputSum)
		r.TokenService.logger.DebugfContext(ctx, "reassign rest [%s] to sender", diff.Decimal())

		var restIdentity []byte
		if transferOpts.RestRecipientIdentity != nil {
			// register it and us it
			if err := wallet.RegisterRecipient(ctx, transferOpts.RestRecipientIdentity); err != nil {
				return nil, nil, errors.WithMessagef(err, "failed to register recipient identity [%s] for the rest, wallet [%s]", transferOpts.RestRecipientIdentity.Identity, wallet.ID())
			}
			restIdentity = transferOpts.RestRecipientIdentity.Identity
		} else {
			restIdentity, err = wallet.GetRecipientIdentity(ctx)
			if err != nil {
				return nil, nil, errors.WithMessagef(err, "failed getting recipient identity for the rest, wallet [%s]", wallet.ID())
			}
		}

		outputTokens = append(outputTokens, &token.Token{
			Owner:    restIdentity,
			Type:     tokenType,
			Quantity: diff.Hex(),
		})
	case -1:
		return nil, nil, errors.Errorf("the sum of the outputs is larger then the sum of the inputs [%s][%s]", inputSum.Decimal(), outputSum.Decimal())
	}

	if r.TokenService.PublicParametersManager().PublicParameters().GraphHiding() {
		r.TokenService.logger.DebugfContext(ctx, "graph hiding enabled, request certification")
		// Check token certification
		cc, err := r.TokenService.CertificationClient()
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "cannot get certification client")
		}
		if err := cc.RequestCertification(ctx, tokenIDs...); err != nil {
			return nil, nil, errors.WithMessagef(err, "failed certifiying inputs")
		}
	}

	return tokenIDs, outputTokens, nil
}

func (r *Request) genOutputs(values []uint64, owners []Identity, tokenType token.Type) ([]*token.Token, token.Quantity, error) {
	pp := r.TokenService.PublicParametersManager().PublicParameters()
	precision := pp.Precision()
	maxTokenValue := pp.MaxTokenValue()
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
			Owner:    owners[i],
			Type:     tokenType,
			Quantity: q.Hex(),
		})
	}
	return outputTokens, outputSum, nil
}

func (r *Request) cleanupInputIDs(ds []*token.ID) []*token.ID {
	newSlice := make([]*token.ID, 0, len(ds))
	for _, item := range ds {
		if item != nil {
			newSlice = append(newSlice, item)
		}
	}
	return newSlice
}
