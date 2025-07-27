/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/protos"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/slices"
	"github.com/pkg/errors"
)

const (
	ProtocolV1 = 1
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

// TokenRequest is a collection of Token Action:
// Issues, to create new Tokens;
// Transfers, to manipulate Tokens (e.g., transfer ownership or redeem)
// The actions in the collection are independent. An action cannot spend tokens created by another action
// in the same Token Request.
// In addition, actions comes with a set of Witnesses to verify the right to spend or the right to issue a given token
type TokenRequest struct {
	Issues            [][]byte
	Transfers         [][]byte
	Signatures        [][]byte
	AuditorSignatures []*AuditorSignature
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

	return &request.TokenRequest{
		Version:    ProtocolV1,
		Actions:    actions,
		Signatures: signatures,
		Auditing:   auditing,
	}, nil
}

func (r *TokenRequest) FromProtos(tr *request.TokenRequest) error {
	// assert version
	if tr.Version != ProtocolV1 {
		return errors.Errorf("invalid token request version, expected [%d], got [%d]", ProtocolV1, tr.Version)
	}

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

func (r *TokenRequest) MarshalToMessageToSign(anchor []byte) ([]byte, error) {
	bytes, err := asn1.Marshal(TokenRequest{Issues: r.Issues, Transfers: r.Transfers})
	if err != nil {
		return nil, errors.Wrapf(err, "audit of tx [%s] failed: error marshal token request for signature", string(anchor))
	}
	return append(bytes, anchor...), nil
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
	ExtraSigners []Identity
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
	return &request.IssueMetadata{
		Issuer:       issuer,
		Inputs:       inputs,
		Outputs:      outputs,
		ExtraSigners: ToProtoIdentitySlice(i.ExtraSigners),
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
	i.ExtraSigners = FromProtoIdentitySlice(issueMetadata.ExtraSigners)
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
	ExtraSigners []Identity
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

	var issuer *request.Identity
	if t.Issuer != nil {
		issuer = &request.Identity{
			Raw: t.Issuer.Bytes(),
		}
	}
	return &request.TransferMetadata{
		Inputs:       inputs,
		Outputs:      outputs,
		ExtraSigners: ToProtoIdentitySlice(t.ExtraSigners),
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
	t.ExtraSigners = FromProtoIdentitySlice(transferMetadata.ExtraSigners)

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

// TokenRequestMetadata is a collection of actions metadata
type TokenRequestMetadata struct {
	// Issues is the list of issue actions metadata
	Issues []*IssueMetadata
	// Transfers is the list of transfer actions metadata
	Transfers []*TransferMetadata
	// Application enables attaching more info to the TokenRequestMetadata
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
