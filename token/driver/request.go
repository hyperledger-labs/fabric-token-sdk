/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"bytes"
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

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

type IssueMetadata struct {
	Issuer     view.Identity
	Outputs    [][]byte
	TokenInfo  [][]byte
	Receivers  []view.Identity
	AuditInfos [][]byte
}

// TransferMetadata contains the following information:
// - For each TokenID there is a sender
type TransferMetadata struct {
	TokenIDs           []*token2.ID
	Outputs            [][]byte
	TokenInfo          [][]byte
	Senders            []view.Identity
	SenderAuditInfos   [][]byte
	Receivers          []view.Identity
	ReceiverIsSender   []bool
	ReceiverAuditInfos [][]byte
}

type TokenRequestMetadata struct {
	Issues      []IssueMetadata
	Transfers   []TransferMetadata
	Application map[string][]byte
}

func (m *TokenRequestMetadata) TokenInfos() [][]byte {
	var res [][]byte
	for _, issue := range m.Issues {
		res = append(res, issue.TokenInfo...)
	}
	for _, transfer := range m.Transfers {
		res = append(res, transfer.TokenInfo...)
	}
	return res
}

func (m *TokenRequestMetadata) GetTokenInfo(tokenRaw []byte) []byte {
	for _, issue := range m.Issues {
		for i, output := range issue.Outputs {
			if bytes.Equal(output, tokenRaw) {
				return issue.TokenInfo[i]
			}
		}
	}
	for _, transfer := range m.Transfers {
		for i, output := range transfer.Outputs {
			if bytes.Equal(output, tokenRaw) {
				return transfer.TokenInfo[i]
			}
		}
	}
	return nil
}

func (m *TokenRequestMetadata) Recipients() [][]byte {
	var res [][]byte
	for _, issue := range m.Issues {
		for _, r := range issue.Receivers {
			res = append(res, r.Bytes())
		}
	}
	for _, transfer := range m.Transfers {
		for _, r := range transfer.Receivers {
			res = append(res, r.Bytes())
		}
	}
	return res
}

func (m *TokenRequestMetadata) Senders() [][]byte {
	var res [][]byte
	for _, transfer := range m.Transfers {
		for _, s := range transfer.Senders {
			res = append(res, s.Bytes())
		}
	}
	return res
}

func (m *TokenRequestMetadata) Issuers() [][]byte {
	var res [][]byte
	for _, issue := range m.Issues {
		res = append(res, issue.Issuer)
	}
	return res
}

func (m *TokenRequestMetadata) Inputs() []*token2.ID {
	var res []*token2.ID
	for _, transfer := range m.Transfers {
		res = append(res, transfer.TokenIDs...)
	}
	return res
}

func (m *TokenRequestMetadata) Bytes() ([]byte, error) {
	meta, err := MarshalMeta(m.Application)
	if err != nil {
		return nil, err
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
			Outputs:            transfer.Outputs,
			TokenInfo:          transfer.TokenInfo,
			Senders:            transfer.Senders,
			SenderAuditInfos:   transfer.SenderAuditInfos,
			Receivers:          transfer.Receivers,
			ReceiverIsSender:   transfer.ReceiverIsSender,
			ReceiverAuditInfos: transfer.ReceiverAuditInfos,
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
		return err
	}

	m.Issues = ser.Issues
	m.Transfers = make([]TransferMetadata, len(ser.Transfers))
	for i, transfer := range ser.Transfers {
		TokenIDs := make([]*token2.ID, len(transfer.TokenIDs))
		for j, tokenID := range transfer.TokenIDs {
			TokenIDs[j] = &token2.ID{
				TxId:  tokenID.TxId,
				Index: uint64(tokenID.Index),
			}
		}
		m.Transfers[i] = TransferMetadata{
			TokenIDs:           TokenIDs,
			Outputs:            transfer.Outputs,
			TokenInfo:          transfer.TokenInfo,
			Senders:            transfer.Senders,
			SenderAuditInfos:   transfer.SenderAuditInfos,
			Receivers:          transfer.Receivers,
			ReceiverIsSender:   transfer.ReceiverIsSender,
			ReceiverAuditInfos: transfer.ReceiverAuditInfos,
		}
	}
	m.Application, err = UnmarshalMeta(ser.Application)
	if err != nil {
		return err
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
	TokenInfo          [][]byte
	Senders            []view.Identity
	SenderAuditInfos   [][]byte
	Receivers          []view.Identity
	ReceiverIsSender   []bool
	ReceiverAuditInfos [][]byte
}

type tokenRequestMetadataSer struct {
	Issues      []IssueMetadata
	Transfers   []TransferMetadataSer
	Application []byte
}
