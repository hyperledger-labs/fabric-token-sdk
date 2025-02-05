/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
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
	return asn1.Marshal(*r)
}

func (r *TokenRequest) FromBytes(raw []byte) error {
	_, err := asn1.Unmarshal(raw, r)
	return err
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
	trm := &request.TokenRequestMetadata{
		Version:     Version,
		Metadata:    nil,
		Application: m.Application,
	}

	// marshal transfers
	for _, transfer := range m.Transfers {
		tokenIDs := make([]*request.TokenID, len(transfer.TokenIDs))
		for j, tokenID := range transfer.TokenIDs {
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

	return proto.Marshal(trm)
}

func (m *TokenRequestMetadata) FromBytes(raw []byte) error {
	trm := &request.TokenRequestMetadata{}
	err := proto.Unmarshal(raw, trm)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal token request metadata")
	}
	m.Application = trm.Application

	// parse metadata
	for _, metadatum := range trm.Metadata {
		im := metadatum.GetIssueMetadata()
		if im != nil {

			m.Issues = append(m.Issues, IssueMetadata{
				Issuer:              im.Issuer.Raw,
				TokenIDs:            TokenIDs,
				OutputsMetadata:     im.OutputsMetadata,
				Receivers:           issue.Receivers,
				ReceiversAuditInfos: im.ReceiversAuditInfos,
				ExtraSigners:        issue.ExtraSigners,
			}
			continue
		}
		tm := metadatum.GetTransferMetadata()
		if tm != nil {

			continue
		}
	}

	// unmarshal transfers

	for i, transfer := range ser.Transfers {
		TokenIDs := make([]*token.ID, len(transfer.TokenIDs))
		for j, tokenID := range transfer.TokenIDs {
			TokenIDs[j] = &token.ID{
				TxId:  tokenID.TxId,
				Index: uint64(tokenID.Index),
			}
		}
		m.Transfers[i] = TransferMetadata{
			TokenIDs:           TokenIDs,
			OutputsAuditInfo:   transfer.OutputAuditInfos,
			OutputsMetadata:    transfer.OutputsMetadata,
			Senders:            transfer.Senders,
			SenderAuditInfos:   transfer.SenderAuditInfos,
			Receivers:          transfer.Receivers,
			ReceiverAuditInfos: transfer.ReceiverAuditInfos,
			ExtraSigners:       transfer.ExtraSigners,
		}
	}

	// unmarshal issues
	m.Issues = make([]IssueMetadata, len(ser.Issues))
	for i, issue := range ser.Issues {
		TokenIDs := make([]*token.ID, len(issue.TokenIDs))
		for j, tokenID := range issue.TokenIDs {
			TokenIDs[j] = &token.ID{
				TxId:  tokenID.TxId,
				Index: uint64(tokenID.Index),
			}
		}
		fmt.Printf("issue with [%d] inputs unmarshalled\n", len(issue.TokenIDs))
		m.Issues[i] = IssueMetadata{
			Issuer:              issue.Issuer,
			TokenIDs:            TokenIDs,
			OutputsMetadata:     issue.OutputsMetadata,
			Receivers:           issue.Receivers,
			ReceiversAuditInfos: issue.ReceiversAuditInfos,
			ExtraSigners:        issue.ExtraSigners,
		}
	}

	// unmarshal application metadata
	m.Application, err = UnmarshalMeta(ser.Application)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal token request metadata: cannot unmarshal application metadata")
	}
	return nil
}

type TokenIDSer struct {
	TxId  string
	Index int
}

type TransferMetadataSer struct {
	TokenIDs           []TokenIDSer
	Outputs            [][]byte
	OutputAuditInfos   [][]byte
	OutputsMetadata    [][]byte
	Senders            []Identity
	SenderAuditInfos   [][]byte
	Receivers          []Identity
	ReceiverAuditInfos [][]byte
	ExtraSigners       []Identity
}

type IssueMetadataSer struct {
	Issuer              Identity
	TokenIDs            []TokenIDSer
	Outputs             [][]byte
	OutputsMetadata     [][]byte
	Receivers           []Identity
	ReceiversAuditInfos [][]byte
	ExtraSigners        []Identity
}

type tokenRequestMetadataSer struct {
	Issues      []IssueMetadataSer
	Transfers   []TransferMetadataSer
	Application []byte
}
