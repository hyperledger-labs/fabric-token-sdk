/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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
	// Senders is the list of senders
	Senders []Identity
	// SendersAuditInfos, for each sender we have audit info to recover the enrollment ID of the sender
	SenderAuditInfos [][]byte

	// OutputsMetadata, for each output we have an OutputsMetadata entry that contains secrets to de-obfuscate the output
	OutputsMetadata [][]byte
	// OutputsAuditInfo, for each output owner we have audit info
	OutputsAuditInfo [][]byte
	// Receivers is the list of receivers
	Receivers []Identity
	// ReceiversAuditInfos, for each receiver we have audit info to recover the enrollment ID of the receiver
	ReceiverAuditInfos [][]byte
	// ReceiverIsSender indicates if the receiver is a sender in this very same action
	ReceiverIsSender []bool

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
	meta, err := MarshalMeta(m.Application)
	if err != nil {
		return nil, errors.New("cannot marshal token request metadata: failed to marshal application metadata")
	}

	transfers := make([]TransferMetadataSer, len(m.Transfers))
	for i, transfer := range m.Transfers {
		TokenIDs := make([]TokenIDSer, len(transfer.TokenIDs))
		for j, tokenID := range transfer.TokenIDs {
			TokenIDs[j].TxId = tokenID.TxId
			TokenIDs[j].Index = int(tokenID.Index)
		}
		transfers[i] = TransferMetadataSer{
			TokenIDs:           TokenIDs,
			OutputAuditInfos:   transfer.OutputsAuditInfo,
			OutputsMetadata:    transfer.OutputsMetadata,
			Senders:            transfer.Senders,
			SenderAuditInfos:   transfer.SenderAuditInfos,
			Receivers:          transfer.Receivers,
			ReceiverIsSender:   transfer.ReceiverIsSender,
			ReceiverAuditInfos: transfer.ReceiverAuditInfos,
			ExtraSigners:       transfer.ExtraSigners,
		}
	}
	ser := tokenRequestMetadataSer{
		Issues:      m.Issues,
		Transfers:   transfers,
		Application: meta,
	}
	return asn1.Marshal(ser)
}

func (m *TokenRequestMetadata) FromBytes(raw []byte) error {
	ser := &tokenRequestMetadataSer{}
	_, err := asn1.Unmarshal(raw, ser)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal token request metadata")
	}

	m.Issues = ser.Issues
	m.Transfers = make([]TransferMetadata, len(ser.Transfers))
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
			ReceiverIsSender:   transfer.ReceiverIsSender,
			ReceiverAuditInfos: transfer.ReceiverAuditInfos,
			ExtraSigners:       transfer.ExtraSigners,
		}
	}
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
	ReceiverIsSender   []bool
	ReceiverAuditInfos [][]byte
	ExtraSigners       []Identity
}

type tokenRequestMetadataSer struct {
	Issues      []IssueMetadata
	Transfers   []TransferMetadataSer
	Application []byte
}
