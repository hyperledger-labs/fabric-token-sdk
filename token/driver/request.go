/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/protos"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	ProtocolV1 = 1
	ProtocolV2 = 2

	// MaxAnchorSize defines the maximum allowed size for anchor parameter in bytes.
	// This limit prevents potential DoS attacks through excessive memory allocation.
	MaxAnchorSize = 128 // bytes
)

// Typed errors for protocol validation
var (
	// ErrAnchorEmpty is returned when the anchor parameter is empty in V2 protocol
	ErrAnchorEmpty = errors.New("anchor cannot be empty")

	// ErrAnchorTooLarge is returned when the anchor exceeds MaxAnchorSize in V2 protocol
	ErrAnchorTooLarge = errors.Errorf("anchor size exceeds maximum allowed size of %d bytes", MaxAnchorSize)

	// ErrUnsupportedVersion is returned when an unsupported protocol version is encountered
	ErrUnsupportedVersion = errors.New("unsupported token request version")

	// ErrInvalidVersion is returned when the protocol version is 0 (invalid)
	ErrInvalidVersion = errors.New("invalid token request: protocol version cannot be 0")

	// ErrVersionBelowMinimum is returned when the protocol version is below the configured minimum
	ErrVersionBelowMinimum = errors.New("token request protocol version is below minimum required version")
)

type (
	// TokenRequestAnchor models the anchor of a token request
	TokenRequestAnchor string
)

type AuditorSignature struct {
	Identity  Identity
	Signature []byte
}

func (r *AuditorSignature) ToProtos() (*request.AuditorSignature, error) {
	return &request.AuditorSignature{
		Identity: &request.Identity{
			Raw: r.Identity,
		},
		Signature: &request.Signature{
			Raw: r.Signature,
		},
	}, nil
}

func (r *AuditorSignature) FromProtos(tr *request.AuditorSignature) error {
	if tr.Identity != nil {
		r.Identity = tr.Identity.Raw
	}
	if tr.Signature != nil {
		r.Signature = tr.Signature.Raw
	}

	return nil
}

// TokenRequest represents a collection of token actions (issuance and transfer).
// Each action within the request is logically independent, though they are processed together.
// A TokenRequest also includes the signatures (witnesses) required to authorize its actions.
type TokenRequest struct {
	Issues            [][]byte
	Transfers         [][]byte
	Signatures        [][]byte
	AuditorSignatures []*AuditorSignature
	// Version specifies the protocol version for this token request.
	// Defaults to ProtocolV2 for new requests.
	// Set to ProtocolV1 when deserializing legacy requests for backward compatibility.
	// The asn1 tag with "-" means this field is never included in ASN.1 marshaling,
	// ensuring backward compatibility with V1 signature verification.
	Version uint32
}

func (r *TokenRequest) Bytes() ([]byte, error) {
	tr, err := r.ToProtos()
	if err != nil {
		return nil, err
	}

	return proto.Marshal(tr)
}

func (r *TokenRequest) FromBytes(raw []byte) error {
	tr := &request.TokenRequest{}
	err := proto.Unmarshal(raw, tr)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling token request")
	}

	return r.FromProtos(tr)
}

func (r *TokenRequest) ToProtos() (*request.TokenRequest, error) {
	actions := append(
		utils.ToActionSlice(request.ActionType_ISSUE, r.Issues),
		utils.ToActionSlice(request.ActionType_TRANSFER, r.Transfers)...,
	)
	signatures := utils.ToSignatureSlice(r.Signatures)
	auditorSignatures, err := protos.ToProtosSlice[request.AuditorSignature, *AuditorSignature](r.AuditorSignatures)
	if err != nil {
		return nil, errors.Wrap(err, "failed converting auditor signatures")
	}
	auditing := &request.Auditing{
		Signatures: auditorSignatures,
	}

	// Use stored version, defaulting to V2 for new requests
	version := r.Version
	if version == 0 {
		version = uint32(ProtocolV2)
	}

	return &request.TokenRequest{
		Version:    version,
		Actions:    actions,
		Signatures: signatures,
		Auditing:   auditing,
	}, nil
}

func (r *TokenRequest) FromProtos(tr *request.TokenRequest) error {
	// Validate version
	if tr.Version != uint32(ProtocolV1) && tr.Version != uint32(ProtocolV2) {
		return errors.Wrapf(ErrUnsupportedVersion, "expected [%d] or [%d], got [%d]", ProtocolV1, ProtocolV2, tr.Version)
	}

	// Store the version from the protobuf
	r.Version = tr.Version

	for _, action := range tr.Actions {
		if action == nil {
			return errors.New("nil action found")
		}
		switch action.Type {
		case request.ActionType_ISSUE:
			r.Issues = append(r.Issues, action.Raw)
		case request.ActionType_TRANSFER:
			r.Transfers = append(r.Transfers, action.Raw)
		default:
			return errors.Errorf("unknown action type [%s]", action.Type)
		}
	}
	for _, signature := range tr.Signatures {
		if signature == nil || len(signature.Raw) == 0 {
			return errors.New("nil signature found")
		}
		r.Signatures = append(r.Signatures, signature.Raw)
	}
	if tr.Auditing != nil {
		r.AuditorSignatures = make([]*AuditorSignature, len(tr.Auditing.Signatures))
		r.AuditorSignatures = slices.GenericSliceOfPointers[AuditorSignature](len(tr.Auditing.Signatures))
		if err := protos.FromProtosSlice(tr.Auditing.Signatures, r.AuditorSignatures); err != nil {
			return errors.Wrap(err, "failed converting auditor signatures")
		}
	}

	return nil
}

// MarshalToMessageToSign creates a canonical byte representation of the TokenRequest
// for signature generation. The behavior depends on the protocol version:
//
// ProtocolV1: Uses simple concatenation (ASN.1-encoded request + anchor).
// This method is maintained for backward compatibility but has known security
// limitations regarding potential hash collisions.
//
// ProtocolV2: Uses structured ASN.1 format with separate Request and Anchor fields,
// providing robust boundary separation and preventing collision attacks.
//
// Parameters:
//   - anchor: A unique identifier (e.g., transaction ID) that binds this signature
//     to a specific context. For V2, must be non-empty and within size limits.
//
// Security considerations:
//   - The anchor MUST be unique per transaction to prevent signature reuse
//   - Signatures are not included in the marshaled data to avoid circular dependencies
//   - V1 uses concatenation which requires careful anchor selection
//   - V2 uses structured format which provides stronger security guarantees
//
// Returns the message bytes to be signed, or an error if marshaling fails.
func (r *TokenRequest) MarshalToMessageToSign(anchor []byte) ([]byte, error) {
	// Dispatch based on protocol version
	switch r.getVersion() {
	case ProtocolV1:
		return r.marshalToMessageToSignV1(anchor)
	case ProtocolV2:
		return r.marshalToMessageToSignV2(anchor)
	default:
		return nil, errors.Errorf("unsupported protocol version [%d]", r.getVersion())
	}
}

// getVersion returns the protocol version of this TokenRequest.
// Returns the stored version, defaulting to V2 for new requests.
func (r *TokenRequest) getVersion() int {
	if r.Version == 0 {
		// Default to V2 for new requests
		return ProtocolV2
	}

	return int(r.Version)
}

// marshalToMessageToSignV1 implements the V1 protocol signature message construction.
// This method maintains the original behavior for backward compatibility with existing
// test data and deployed systems.
//
// WARNING: This implementation has known security limitations:
//   - Simple concatenation without delimiter allows potential boundary ambiguity
//   - Different (request, anchor) pairs could theoretically produce identical messages
//
// This method is preserved unchanged to ensure regression tests pass.
func (r *TokenRequest) marshalToMessageToSignV1(anchor []byte) ([]byte, error) {
	// Use a struct that matches the original TokenRequest structure (4 fields).
	// Even though only Issues and Transfers are populated, ASN.1 encodes all fields,
	// including empty Signatures and AuditorSignatures as empty sequences.
	// This ensures identical ASN.1 encoding for backward compatibility with V1 signatures.
	type tokenRequestV1 struct {
		Issues            [][]byte
		Transfers         [][]byte
		Signatures        [][]byte
		AuditorSignatures []*AuditorSignature
	}

	bytes, err := asn1.Marshal(tokenRequestV1{
		Issues:    r.Issues,
		Transfers: r.Transfers,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "audit of tx [%s] failed: error marshal token request for signature", string(anchor))
	}

	return append(bytes, anchor...), nil
}

// marshalToMessageToSignV2 implements the V2 protocol signature message construction
// using a secure structured ASN.1 format that prevents hash collision vulnerabilities.
//
// Security improvements over V1:
//   - Structured ASN.1 format with explicit Request and Anchor fields
//   - Clear boundary separation prevents collision attacks
//   - Input validation ensures anchor meets security requirements
//   - Hex-encoded error messages prevent sensitive data exposure
//
// This method should be used for all new token requests to benefit from
// enhanced security properties.
//
// This implementation uses an optimized fast marshaller that avoids reflection overhead
// while maintaining full ASN.1 compatibility.
func (r *TokenRequest) marshalToMessageToSignV2(anchor []byte) ([]byte, error) {
	// Input validation with typed errors
	if len(anchor) == 0 {
		return nil, ErrAnchorEmpty
	}
	if len(anchor) > MaxAnchorSize {
		return nil, ErrAnchorTooLarge
	}

	// Marshal the request data using fast marshaller (Issues and Transfers only)
	requestBytes, err := fastMarshalTokenRequestForSigning(r.Issues, r.Transfers)
	if err != nil {
		return nil, errors.Wrapf(err, "audit of tx [%x] failed: error marshal token request for signature", anchor)
	}

	// Marshal the complete structured message using fast marshaller
	msgBytes, err := fastMarshalSignatureMessageV2(requestBytes, anchor)
	if err != nil {
		return nil, errors.Wrapf(err, "audit of tx [%x] failed: error marshal signature message", anchor)
	}

	return msgBytes, nil
}

type AuditableIdentity struct {
	Identity  Identity
	AuditInfo []byte
}

func (a *AuditableIdentity) ToProtos() (*request.AuditableIdentity, error) {
	return &request.AuditableIdentity{
		Identity: &request.Identity{
			Raw: a.Identity,
		},
		AuditInfo: a.AuditInfo,
	}, nil
}

func (a *AuditableIdentity) FromProtos(auditableIdentity *request.AuditableIdentity) error {
	a.Identity = ToIdentity(auditableIdentity.Identity)
	a.AuditInfo = auditableIdentity.AuditInfo

	return nil
}

type IssueInputMetadata struct {
	TokenID *token.ID
}

func (i *IssueInputMetadata) ToProtos() (*request.IssueInputMetadata, error) {
	tokenID, err := utils.ToTokenID(i.TokenID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal token ID [%s]", i.TokenID)
	}

	return &request.IssueInputMetadata{
		TokenId: tokenID,
	}, nil
}

func (i *IssueInputMetadata) FromProtos(issueInputMetadata *request.IssueInputMetadata) error {
	i.TokenID = ToTokenID(issueInputMetadata.TokenId)

	return nil
}

// IssueOutputMetadata is the metadata of an output in an issue action
type IssueOutputMetadata struct {
	OutputMetadata []byte
	// Receivers, for each output we have a receiver
	Receivers []*AuditableIdentity
}

func (i *IssueOutputMetadata) ToProtos() (*request.OutputMetadata, error) {
	receivers, err := protos.ToProtosSlice[request.AuditableIdentity, *AuditableIdentity](i.Receivers)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling receivers")
	}

	return &request.OutputMetadata{
		Metadata:  i.OutputMetadata,
		Receivers: receivers,
	}, nil
}

func (i *IssueOutputMetadata) FromProtos(outputsMetadata *request.OutputMetadata) error {
	if outputsMetadata == nil {
		return nil
	}
	i.OutputMetadata = outputsMetadata.Metadata
	i.Receivers = slices.GenericSliceOfPointers[AuditableIdentity](len(outputsMetadata.Receivers))
	if err := protos.FromProtosSlice(outputsMetadata.Receivers, i.Receivers); err != nil {
		return errors.Wrap(err, "failed unmarshalling receivers metadata")
	}

	return nil
}

func (i *IssueOutputMetadata) RecipientAt(index int) *AuditableIdentity {
	if index < 0 || index >= len(i.Receivers) {
		return nil
	}

	return i.Receivers[index]
}

// IssueMetadata contains the metadata of an issue action.
// In more details, there is an issuer and a list of outputs.
// For each output, there is a token info and a list of receivers with their audit info to recover their enrollment ID.
type IssueMetadata struct {
	// Issuer is the identity of the issuer
	Issuer  AuditableIdentity
	Inputs  []*IssueInputMetadata
	Outputs []*IssueOutputMetadata
	// ExtraSigners is the list of extra identities that are not part of the issue action per se
	// but needs to sign the request
	ExtraSigners []*AuditableIdentity
}

func (i *IssueMetadata) ToProtos() (*request.IssueMetadata, error) {
	issuer, err := i.Issuer.ToProtos()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling issuer [%v]", i.Issuer)
	}
	inputs, err := protos.ToProtosSlice(i.Inputs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling inputs")
	}
	outputs, err := protos.ToProtosSlice(i.Outputs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling outputs")
	}

	extraSigners, err := protos.ToProtosSlice[request.AuditableIdentity, *AuditableIdentity](i.ExtraSigners)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling extra signers")
	}

	return &request.IssueMetadata{
		Issuer:       issuer,
		Inputs:       inputs,
		Outputs:      outputs,
		ExtraSigners: extraSigners,
	}, nil
}

func (i *IssueMetadata) FromProtos(issueMetadata *request.IssueMetadata) error {
	issuer := &AuditableIdentity{}
	if err := issuer.FromProtos(issueMetadata.Issuer); err != nil {
		return errors.Wrapf(err, "failed unmarshalling issuer [%v]", issueMetadata.Issuer)
	}
	i.Issuer = *issuer
	i.Inputs = slices.GenericSliceOfPointers[IssueInputMetadata](len(issueMetadata.Inputs))
	err := protos.FromProtosSlice[request.IssueInputMetadata, *IssueInputMetadata](issueMetadata.Inputs, i.Inputs)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling input metadata")
	}
	i.Outputs = slices.GenericSliceOfPointers[IssueOutputMetadata](len(issueMetadata.Outputs))
	err = protos.FromProtosSlice(issueMetadata.Outputs, i.Outputs)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling output metadata")
	}
	i.ExtraSigners = slices.GenericSliceOfPointers[AuditableIdentity](len(issueMetadata.ExtraSigners))
	if err = protos.FromProtosSlice[request.AuditableIdentity, *AuditableIdentity](issueMetadata.ExtraSigners, i.ExtraSigners); err != nil {
		return errors.Wrap(err, "failed unmarshalling extra signers")
	}

	return nil
}

func (i *IssueMetadata) Receivers() []Identity {
	res := make([]Identity, 0, len(i.Outputs))
	for _, output := range i.Outputs {
		for _, receiver := range output.Receivers {
			if receiver == nil {
				res = append(res, nil)
			} else {
				res = append(res, receiver.Identity)
			}
		}
	}

	return res
}

type TransferInputMetadata struct {
	TokenID *token.ID
	Senders []*AuditableIdentity
}

func (t *TransferInputMetadata) ToProtos() (*request.TransferInputMetadata, error) {
	senders, err := protos.ToProtosSlice(t.Senders)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling senders")
	}
	tokenID, err := utils.ToTokenID(t.TokenID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling tokenID")
	}

	return &request.TransferInputMetadata{
		TokenId: tokenID,
		Senders: senders,
	}, nil
}

func (t *TransferInputMetadata) FromProtos(transferInputMetadata *request.TransferInputMetadata) error {
	if transferInputMetadata == nil {
		return nil
	}
	t.TokenID = ToTokenID(transferInputMetadata.TokenId)
	t.Senders = slices.GenericSliceOfPointers[AuditableIdentity](len(transferInputMetadata.Senders))
	if err := protos.FromProtosSlice(transferInputMetadata.Senders, t.Senders); err != nil {
		return errors.Wrap(err, "failed unmarshalling token metadata")
	}

	return nil
}

// TransferOutputMetadata is the metadata of an output in a transfer action
type TransferOutputMetadata struct {
	OutputMetadata []byte
	// OutputAuditInfo, for each output owner we have audit info
	OutputAuditInfo []byte
	// Receivers is the list of receivers
	Receivers []*AuditableIdentity
}

func (t *TransferOutputMetadata) ToProtos() (*request.OutputMetadata, error) {
	receivers, err := protos.ToProtosSlice(t.Receivers)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling receivers")
	}

	return &request.OutputMetadata{
		Metadata:  t.OutputMetadata,
		AuditInfo: t.OutputAuditInfo,
		Receivers: receivers,
	}, nil
}

func (t *TransferOutputMetadata) FromProtos(transferOutputMetadata *request.OutputMetadata) error {
	if transferOutputMetadata == nil {
		return nil
	}
	t.OutputMetadata = transferOutputMetadata.Metadata
	t.OutputAuditInfo = transferOutputMetadata.AuditInfo
	t.Receivers = slices.GenericSliceOfPointers[AuditableIdentity](len(transferOutputMetadata.Receivers))
	if err := protos.FromProtosSlice(transferOutputMetadata.Receivers, t.Receivers); err != nil {
		return errors.Wrap(err, "failed unmarshalling receivers metadata")
	}

	return nil
}

func (t *TransferOutputMetadata) RecipientAt(index int) *AuditableIdentity {
	if index < 0 || index >= len(t.Receivers) {
		return nil
	}

	return t.Receivers[index]
}

// TransferMetadata contains the metadata of a transfer action
// For each TokenID there is a sender with its audit info to recover its enrollment ID,
// For each Output there is:
// - A OutputMetadata entry to de-obfuscate the output;
// - A Receiver identity;
// - A ReceiverAuditInfo entry to recover the enrollment ID of the receiver
// - A Flag to indicate if the receiver is a sender in this very same action
type TransferMetadata struct {
	Inputs  []*TransferInputMetadata
	Outputs []*TransferOutputMetadata
	// ExtraSigners is the list of extra identities that are not part of the transfer action per se
	// but needs to sign the request
	ExtraSigners []*AuditableIdentity
	// Issuer contains the identity of the issuer to sign the transfer action
	Issuer Identity
}

// TokenIDAt returns the TokenID at the given index.
// It returns nil if the index is out of bounds.
func (t *TransferMetadata) TokenIDAt(index int) *token.ID {
	if index < 0 || index >= len(t.Inputs) {
		return nil
	}

	return t.Inputs[index].TokenID
}

func (t *TransferMetadata) ToProtos() (*request.TransferMetadata, error) {
	inputs, err := protos.ToProtosSlice[request.TransferInputMetadata, *TransferInputMetadata](t.Inputs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling inputs")
	}
	outputs, err := protos.ToProtosSlice(t.Outputs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling outputs")
	}
	extraSigners, err := protos.ToProtosSlice[request.AuditableIdentity, *AuditableIdentity](t.ExtraSigners)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling extra signers")
	}

	var issuer *request.Identity
	if t.Issuer != nil {
		issuer = &request.Identity{
			Raw: t.Issuer.Bytes(),
		}
	}

	return &request.TransferMetadata{
		Inputs:       inputs,
		Outputs:      outputs,
		ExtraSigners: extraSigners,
		Issuer:       issuer,
	}, nil
}

func (t *TransferMetadata) FromProtos(transferMetadata *request.TransferMetadata) error {
	t.Inputs = slices.GenericSliceOfPointers[TransferInputMetadata](len(transferMetadata.Inputs))
	if err := protos.FromProtosSlice(transferMetadata.Inputs, t.Inputs); err != nil {
		return errors.Wrap(err, "failed unmarshalling inputs")
	}
	t.Outputs = slices.GenericSliceOfPointers[TransferOutputMetadata](len(transferMetadata.Outputs))
	if err := protos.FromProtosSlice(transferMetadata.Outputs, t.Outputs); err != nil {
		return errors.Wrap(err, "failed unmarshalling outputs")
	}
	t.ExtraSigners = slices.GenericSliceOfPointers[AuditableIdentity](len(transferMetadata.ExtraSigners))
	if err := protos.FromProtosSlice[request.AuditableIdentity, *AuditableIdentity](transferMetadata.ExtraSigners, t.ExtraSigners); err != nil {
		return errors.Wrap(err, "failed unmarshalling extra signers")
	}

	t.Issuer = nil
	if transferMetadata.Issuer != nil {
		t.Issuer = transferMetadata.Issuer.Raw
	}

	return nil
}

func (t *TransferMetadata) Receivers() []Identity {
	res := make([]Identity, 0, len(t.Outputs))
	for _, output := range t.Outputs {
		for _, receiver := range output.Receivers {
			if receiver == nil {
				res = append(res, nil)
			} else {
				res = append(res, receiver.Identity)
			}
		}
	}

	return res
}

func (t *TransferMetadata) Senders() []Identity {
	res := make([]Identity, 0, len(t.Inputs))
	for _, output := range t.Inputs {
		for _, sender := range output.Senders {
			if sender == nil {
				res = append(res, nil)
			} else {
				res = append(res, sender.Identity)
			}
		}
	}

	return res
}

func (t *TransferMetadata) TokenIDs() []*token.ID {
	res := make([]*token.ID, 0, len(t.Inputs))
	for _, input := range t.Inputs {
		if input == nil {
			continue
		}
		res = append(res, input.TokenID)
	}

	return res
}

// TokenRequestMetadata contains the supplementary information needed to process and interpret a TokenRequest.
// It includes metadata for each issuance and transfer action, enabling de-obfuscation and identity recovery.
type TokenRequestMetadata struct {
	// Issues contains metadata for each issuance action in the corresponding TokenRequest.
	Issues []*IssueMetadata
	// Transfers contains metadata for each transfer action in the corresponding TokenRequest.
	Transfers []*TransferMetadata
	// Application allows for attaching arbitrary application-level metadata to the token request.
	Application map[string][]byte
}

func (m *TokenRequestMetadata) Bytes() ([]byte, error) {
	trm, err := m.ToProtos()
	if err != nil {
		return nil, err
	}

	return proto.Marshal(trm)
}

func (m *TokenRequestMetadata) FromBytes(raw []byte) error {
	trm := &request.TokenRequestMetadata{}
	err := proto.Unmarshal(raw, trm)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal token request metadata")
	}

	return m.FromProtos(trm)
}

func (m *TokenRequestMetadata) ToProtos() (*request.TokenRequestMetadata, error) {
	trm := &request.TokenRequestMetadata{
		Version:     ProtocolV1,
		Metadata:    nil,
		Application: m.Application,
	}
	trm.Metadata = make([]*request.ActionMetadata, 0, len(m.Issues)+len(m.Transfers))
	for _, meta := range m.Issues {
		if meta == nil {
			return nil, errors.Errorf("failed unmarshalling issue metadata, it is nil")
		}
		metaProto, err := meta.ToProtos()
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshalling issue metadata")
		}
		trm.Metadata = append(trm.Metadata, &request.ActionMetadata{
			Metadata: &request.ActionMetadata_IssueMetadata{
				IssueMetadata: metaProto,
			},
		})
	}
	for _, meta := range m.Transfers {
		if meta == nil {
			return nil, errors.Errorf("failed unmarshalling issue metadata, it is nil")
		}
		metaProto, err := meta.ToProtos()
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshalling transfer metadata")
		}
		trm.Metadata = append(trm.Metadata, &request.ActionMetadata{
			Metadata: &request.ActionMetadata_TransferMetadata{
				TransferMetadata: metaProto,
			},
		})
	}

	return trm, nil
}

func (m *TokenRequestMetadata) FromProtos(trm *request.TokenRequestMetadata) error {
	// assert version
	if trm.Version != ProtocolV1 {
		return errors.Errorf("invalid token request metadata version, expected [%d], got [%d]", ProtocolV1, trm.Version)
	}

	m.Application = trm.Application
	for _, meta := range trm.Metadata {
		im := meta.GetIssueMetadata()
		if im != nil {
			issueMetadata := &IssueMetadata{}
			if err := issueMetadata.FromProtos(im); err != nil {
				return errors.Wrapf(err, "failed unmarshalling issue metadata")
			}
			m.Issues = append(m.Issues, issueMetadata)

			continue
		}
		tm := meta.GetTransferMetadata()
		if tm != nil {
			transferMetadata := &TransferMetadata{}
			if err := transferMetadata.FromProtos(tm); err != nil {
				return errors.Wrapf(err, "failed unmarshalling transfer metadata")
			}
			m.Transfers = append(m.Transfers, transferMetadata)

			continue
		}

		return errors.Errorf("failed unmarshalling metadata, type not recognized")
	}

	return nil
}
