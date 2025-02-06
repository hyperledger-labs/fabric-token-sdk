/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	Version = 1
)

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
	AuditorSignatures [][]byte
}

func (r *TokenRequest) Bytes() ([]byte, error) {
	tr, err := r.ToProtos()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(tr)
}

func (r *TokenRequest) ToProtos() (*request.TokenRequest, error) {
	tr := &request.TokenRequest{
		Version: Version,
	}
	for _, issue := range r.Issues {
		tr.Actions = append(tr.Actions,
			&request.Action{
				Type: request.ActionType_ISSUE,
				Raw:  issue,
			},
		)
	}
	for _, transfer := range r.Transfers {
		tr.Actions = append(tr.Actions,
			&request.Action{
				Type: request.ActionType_TRANSFER,
				Raw:  transfer,
			},
		)
	}
	for _, signature := range r.Signatures {
		tr.Signatures = append(tr.Signatures, &request.Signature{
			Raw: signature,
		})
	}
	for _, signature := range r.AuditorSignatures {
		tr.AuditorSignatures = append(tr.AuditorSignatures, &request.Signature{
			Raw: signature,
		})
	}

	return tr, nil
}

func (r *TokenRequest) FromBytes(raw []byte) error {
	tr := &request.TokenRequest{}
	err := proto.Unmarshal(raw, tr)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling token request")
	}
	return r.FromProtos(tr)
}

func (r *TokenRequest) FromProtos(tr *request.TokenRequest) error {
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
	for _, signature := range tr.AuditorSignatures {
		if signature == nil || len(signature.Raw) == 0 {
			return errors.New("nil auditor signature found")
		}
		r.AuditorSignatures = append(r.AuditorSignatures, signature.Raw)
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

// IssueMetadata contains the metadata of an issue action.
// In more details, there is an issuer and a list of outputs.
// For each output, there is a token info and a list of receivers with their audit info to recover their enrollment ID.
type IssueMetadata struct {
	// Issuer is the identity of the issuer
	Issuer Identity

	// TokenIDs is the list of TokenIDs spent by this action
	TokenIDs []*token.ID

	// OutputsMetadata, for each output we have a OutputsMetadata entry that contains secrets to de-obfuscate the output
	OutputsMetadata [][]byte
	// Receivers, for each output we have a receiver
	Receivers []Identity
	// ReceiversAuditInfos, for each receiver we have audit info to recover the enrollment ID of the receiver
	ReceiversAuditInfos [][]byte

	// ExtraSigners is the list of extra identities that are not part of the issue action per se
	// but needs to sign the request
	ExtraSigners []Identity
}

// TransferMetadata contains the metadata of a transfer action
// For each TokenID there is a sender with its audit info to recover its enrollment ID,
// For each Output there is:
// - A OutputsMetadata entry to de-obfuscate the output;
// - A Receiver identity;
// - A ReceiverAuditInfo entry to recover the enrollment ID of the receiver
// - A Flag to indicate if the receiver is a sender in this very same action
type TransferMetadata struct {
	// TokenIDs is the list of TokenIDs spent by this action
	TokenIDs []*token.ID
	// Senders is the list of senders.
	// For each input, a sender is the input's owner
	Senders []Identity
	// SendersAuditInfos, for each sender we have audit info
	SenderAuditInfos [][]byte

	// OutputsMetadata, for each output we have an OutputsMetadata entry that contains secrets to de-obfuscate the output
	OutputsMetadata [][]byte
	// OutputsAuditInfo, for each output owner we have audit info
	OutputsAuditInfo [][]byte
	// Receivers is the list of receivers
	Receivers []Identity
	// ReceiversAuditInfos, for each receiver we have audit info to recover the enrollment ID of the receiver
	ReceiverAuditInfos [][]byte

	// ExtraSigners is the list of extra identities that are not part of the transfer action per se
	// but needs to sign the request
	ExtraSigners []Identity
}

// TokenIDAt returns the TokenID at the given index.
// It returns nil if the index is out of bounds.
func (tm *TransferMetadata) TokenIDAt(index int) *token.ID {
	if index < 0 || index >= len(tm.TokenIDs) {
		return nil
	}
	return tm.TokenIDs[index]
}

// TokenRequestMetadata is a collection of actions metadata
type TokenRequestMetadata struct {
	// Issues is the list of issue actions metadata
	Issues []IssueMetadata
	// Transfers is the list of transfer actions metadata
	Transfers []TransferMetadata
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

func (m *TokenRequestMetadata) ToProtos() (*request.TokenRequestMetadata, error) {
	trm := &request.TokenRequestMetadata{
		Version:     Version,
		Metadata:    nil,
		Application: m.Application,
	}

	// marshal transfers
	for _, transfer := range m.Transfers {
		tokenIDs := make([]*request.TokenID, len(transfer.TokenIDs))
		for j, tokenID := range transfer.TokenIDs {
			if tokenID == nil {
				return nil, errors.Errorf("nil token ID at index [%d]", j)
			}
			tokenIDs[j] = &request.TokenID{}
			tokenIDs[j].TxId = tokenID.TxId
			tokenIDs[j].Index = tokenID.Index
		}
		senders := make([]*request.Identity, len(transfer.Senders))
		for j, sender := range transfer.Senders {
			senders[j] = &request.Identity{
				Raw: sender,
			}
		}
		receivers := make([]*request.Identity, len(transfer.Receivers))
		for j, receiver := range transfer.Receivers {
			receivers[j] = &request.Identity{
				Raw: receiver,
			}
		}
		extraSigners := make([]*request.Identity, len(transfer.ExtraSigners))
		for j, extraSigner := range transfer.ExtraSigners {
			extraSigners[j] = &request.Identity{
				Raw: extraSigner,
			}
		}
		trm.Metadata = append(trm.Metadata, &request.ActionMetadata{
			Metadata: &request.ActionMetadata_TransferMetadata{
				TransferMetadata: &request.TransferMetadata{
					TokenIds:           tokenIDs,
					OutputAudit_Infos:  transfer.OutputsAuditInfo,
					OutputsMetadata:    transfer.OutputsMetadata,
					Senders:            senders,
					SenderAuditInfos:   transfer.SenderAuditInfos,
					Receivers:          receivers,
					ReceiverAuditInfos: transfer.ReceiverAuditInfos,
					ExtraSigners:       extraSigners,
				},
			},
		})
	}

	// marshal issues
	for _, issue := range m.Issues {
		tokenIDs := make([]*request.TokenID, len(issue.TokenIDs))
		for j, tokenID := range issue.TokenIDs {
			if tokenID == nil {
				return nil, errors.Errorf("nil token ID at index [%d]", j)
			}
			tokenIDs[j] = &request.TokenID{}
			tokenIDs[j].TxId = tokenID.TxId
			tokenIDs[j].Index = tokenID.Index
		}
		receivers := make([]*request.Identity, len(issue.Receivers))
		for j, receiver := range issue.Receivers {
			receivers[j] = &request.Identity{
				Raw: receiver,
			}
		}
		extraSigners := make([]*request.Identity, len(issue.ExtraSigners))
		for j, extraSigner := range issue.ExtraSigners {
			extraSigners[j] = &request.Identity{
				Raw: extraSigner,
			}
		}
		trm.Metadata = append(trm.Metadata, &request.ActionMetadata{
			Metadata: &request.ActionMetadata_IssueMetadata{
				IssueMetadata: &request.IssueMetadata{
					Issuer: &request.Identity{
						Raw: issue.Issuer,
					},
					TokenIds:           tokenIDs,
					OutputsMetadata:    issue.OutputsMetadata,
					Receivers:          receivers,
					ReceiverAuditInfos: issue.ReceiversAuditInfos,
					ExtraSigners:       extraSigners,
				},
			},
		})
	}

	return trm, nil
}

func (m *TokenRequestMetadata) FromBytes(raw []byte) error {
	trm := &request.TokenRequestMetadata{}
	err := proto.Unmarshal(raw, trm)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal token request metadata")
	}

	return m.FromProtos(trm)
}

func (m *TokenRequestMetadata) FromProtos(trm *request.TokenRequestMetadata) error {
	m.Application = trm.Application
	for _, metadatum := range trm.Metadata {
		im := metadatum.GetIssueMetadata()
		if im != nil {
			tokenIDs := make([]*token.ID, len(im.TokenIds))
			for j, tokenID := range im.TokenIds {
				if tokenID != nil {
					tokenIDs[j] = &token.ID{}
					tokenIDs[j].TxId = tokenID.TxId
					tokenIDs[j].Index = tokenID.Index
				}
			}
			receivers := make([]Identity, len(im.Receivers))
			for j, receiver := range im.Receivers {
				if receiver != nil {
					receivers[j] = receiver.Raw
				}
			}
			extraSigners := make([]Identity, len(im.ExtraSigners))
			for j, extraSigner := range im.ExtraSigners {
				if extraSigner != nil {
					extraSigners[j] = extraSigner.Raw
				}
			}
			m.Issues = append(m.Issues, IssueMetadata{
				Issuer:              im.Issuer.Raw,
				TokenIDs:            tokenIDs,
				OutputsMetadata:     im.OutputsMetadata,
				Receivers:           receivers,
				ReceiversAuditInfos: im.ReceiverAuditInfos,
				ExtraSigners:        extraSigners,
			})
			continue
		}

		tm := metadatum.GetTransferMetadata()
		if tm != nil {
			tokenIDs := make([]*token.ID, len(tm.TokenIds))
			for j, tokenID := range tm.TokenIds {
				if tokenID != nil {
					tokenIDs[j] = &token.ID{}
					tokenIDs[j].TxId = tokenID.TxId
					tokenIDs[j].Index = tokenID.Index
				}
			}
			senders := make([]Identity, len(tm.Senders))
			for j, sender := range tm.Senders {
				if sender != nil {
					senders[j] = sender.Raw
				}
			}
			receivers := make([]Identity, len(tm.Receivers))
			for j, receiver := range tm.Receivers {
				if receiver != nil {
					receivers[j] = receiver.Raw
				}
			}
			extraSigners := make([]Identity, len(tm.ExtraSigners))
			for j, extraSigner := range tm.ExtraSigners {
				if extraSigner != nil {
					extraSigners[j] = extraSigner.Raw
				}
			}

			m.Transfers = append(m.Transfers, TransferMetadata{
				TokenIDs:           tokenIDs,
				Senders:            senders,
				SenderAuditInfos:   tm.SenderAuditInfos,
				OutputsMetadata:    tm.OutputsMetadata,
				OutputsAuditInfo:   tm.OutputAudit_Infos,
				Receivers:          receivers,
				ReceiverAuditInfos: tm.ReceiverAuditInfos,
				ExtraSigners:       extraSigners,
			})
			continue
		}
	}
	return nil
}
