/*
 *
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 * /
 *
 */

package token

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type tms interface {
	// DeserializeToken returns the token and its issuer (if any).
	DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, view.Identity, error)
	// GetEnrollmentID extracts the enrollment id from the passed audit information
	GetEnrollmentID(bytes []byte) (string, error)
}

// Metadata contains the metadata of a Token Request
type Metadata struct {
	tms                  tms
	tokenRequestMetadata *api2.TokenRequestMetadata
}

// GetToken unmarshals the given bytes to extract the token and its issuer (if any).
func (m *Metadata) GetToken(raw []byte) (*token2.Token, view.Identity, []byte, error) {
	tokenInfoRaw := m.tokenRequestMetadata.GetTokenInfo(raw)
	if len(tokenInfoRaw) == 0 {
		logger.Debugf("metadata for [%s] not found", hash.Hashable(raw).String())
		return nil, nil, nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(raw).String())
	}
	tok, id, err := m.tms.DeserializeToken(raw, tokenInfoRaw)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed getting token in the clear")
	}
	return tok, id, tokenInfoRaw, nil
}

// SpentTokenID returns the token IDs of the tokens that were spent by the Token Request this metadata is associated with.
func (m *Metadata) SpentTokenID() []*token2.ID {
	var res []*token2.ID
	for _, transfer := range m.tokenRequestMetadata.Transfers {
		res = append(res, transfer.TokenIDs...)
	}
	return res
}

// FilterBy returns a new Metadata containing only the metadata that matches the given enrollment IDs.
// For Issue actions: the returned metadata will contain only the outputs whose owner has the given enrollment IDs.
// For Transfer actions:
// - The returned metadata will contain only the outputs whose owner has the given enrollment IDs;
// - The senders are all included;
// - The list of token IDs will be empty.
func (m *Metadata) FilterBy(eIDs ...string) (*Metadata, error) {
	res := &Metadata{
		tms:                  m.tms,
		tokenRequestMetadata: &api2.TokenRequestMetadata{},
	}

	// filter issues
	for _, issue := range m.tokenRequestMetadata.Issues {
		// By default, an issue action contain only one receiver

		// If the receiver has the given enrollment ID, add it
		recipientEID, err := m.tms.GetEnrollmentID(issue.ReceiversAuditInfos[0])
		if err != nil {
			return nil, errors.Wrap(err, "failed getting enrollment ID")
		}
		if search(eIDs, recipientEID) == -1 {
			logger.Debugf("skipping issue for [%s]", recipientEID)
			continue
		}
		res.tokenRequestMetadata.Issues = append(res.tokenRequestMetadata.Issues, issue)
	}

	// filter transfers
	for _, transfer := range m.tokenRequestMetadata.Transfers {
		transferRes := api2.TransferMetadata{}

		// Filter senders:
		// All senders are appended
		for i, sender := range transfer.Senders {
			transferRes.Senders = append(transferRes.Senders, sender)
			transferRes.SenderAuditInfos = append(transferRes.SenderAuditInfos, transfer.SenderAuditInfos[i])
		}

		// Filter outputs
		// if the receiver has the given enrollment ID, add it. Otherwise, add empty entries
		for i, auditInfo := range transfer.ReceiverAuditInfos {
			recipientEID, err := m.tms.GetEnrollmentID(auditInfo)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting enrollment ID")
			}
			var Outputs []byte
			var TokenInfo []byte
			var Receivers view.Identity
			var ReceiverIsSender bool
			var ReceiverAuditInfos []byte

			if search(eIDs, recipientEID) != -1 {
				logger.Debugf("keeping transfer for [%s]", recipientEID)
				Outputs = transfer.Outputs[i]
				TokenInfo = transfer.TokenInfo[i]
				Receivers = transfer.Receivers[i]
				ReceiverIsSender = transfer.ReceiverIsSender[i]
				ReceiverAuditInfos = transfer.ReceiverAuditInfos[i]
			} else {
				logger.Debugf("skipping transfer for [%s]", recipientEID)
			}

			transferRes.Outputs = append(transferRes.Outputs, Outputs)
			transferRes.Receivers = append(transferRes.Receivers, Receivers)
			transferRes.ReceiverIsSender = append(transferRes.ReceiverIsSender, ReceiverIsSender)
			transferRes.ReceiverAuditInfos = append(transferRes.ReceiverAuditInfos, ReceiverAuditInfos)
			transferRes.TokenInfo = append(transferRes.TokenInfo, TokenInfo)
		}
		logger.Debugf("keeping transfer with [%d] out of [%d] outputs", len(transferRes.Outputs), len(transfer.Outputs))
		res.tokenRequestMetadata.Transfers = append(res.tokenRequestMetadata.Transfers, transferRes)
	}

	logger.Debugf("filtered metadata for [% x] from [%d:%d] to [%d:%d]",
		eIDs,
		len(m.tokenRequestMetadata.Issues), len(m.tokenRequestMetadata.Transfers),
		len(res.tokenRequestMetadata.Issues), len(res.tokenRequestMetadata.Transfers))

	return res, nil
}

// Issue returns the i-th issue metadata, if present
func (m *Metadata) Issue(i int) (*IssueMetadata, error) {
	if i >= len(m.tokenRequestMetadata.Issues) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.tokenRequestMetadata.Issues))
	}
	return &IssueMetadata{IssueMetadata: &m.tokenRequestMetadata.Issues[i]}, nil
}

// Transfer returns the i-th transfer metadata, if present
func (m *Metadata) Transfer(i int) (*TransferMetadata, error) {
	if i >= len(m.tokenRequestMetadata.Transfers) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.tokenRequestMetadata.Transfers))
	}
	return &TransferMetadata{TransferMetadata: &m.tokenRequestMetadata.Transfers[i]}, nil
}

// IssueMetadata contains the metadata of an issue action
type IssueMetadata struct {
	*api2.IssueMetadata
}

// Match returns true if the given action matches this metadata
func (m *IssueMetadata) Match(action *IssueAction) error {
	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}
	if len(m.Outputs) != len(m.TokenInfo) {
		return errors.Errorf("expected [%d] token info but got [%d]", len(m.Outputs), len(m.TokenInfo))
	}
	if len(m.Outputs) != len(m.Receivers) {
		return errors.Errorf("expected [%d] receivers but got [%d]", len(m.Outputs), len(m.Receivers))
	}
	if len(m.Outputs) != len(m.ReceiversAuditInfos) {
		return errors.Errorf("expected [%d] receiver audit infos but got [%d]", len(m.Outputs), len(m.ReceiversAuditInfos))
	}
	return nil
}

// IsOutputAbsent returns true if the given output's metadata is absent
func (m *IssueMetadata) IsOutputAbsent(j int) bool {
	return len(m.TokenInfo[j]) == 0
}

// TransferMetadata contains the metadata of a transfer action
type TransferMetadata struct {
	*api2.TransferMetadata
}

// Match returns true if the given action matches this metadata
func (m *TransferMetadata) Match(action *TransferAction) error {
	if len(m.TokenIDs) != 0 && len(m.TokenIDs) != len(m.Senders) {
		return errors.Errorf("expected [%d] token IDs and senders but got [%d]", len(m.TokenIDs), len(m.Senders))
	}
	if len(m.Senders) != len(m.SenderAuditInfos) {
		return errors.Errorf("expected [%d] senders and sender audit infos but got [%d]", len(m.Senders), len(m.SenderAuditInfos))
	}

	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}
	if len(m.Outputs) != len(m.TokenInfo) {
		return errors.Errorf("expected [%d] token info but got [%d]", len(m.Outputs), len(m.TokenInfo))
	}
	if len(m.Outputs) != len(m.Receivers) {
		return errors.Errorf("expected [%d] receivers but got [%d]", len(m.Outputs), len(m.Receivers))
	}
	if len(m.Outputs) != len(m.ReceiverAuditInfos) {
		return errors.Errorf("expected [%d] receiver audit infos but got [%d]", len(m.Outputs), len(m.ReceiverAuditInfos))
	}
	if len(m.Outputs) != len(m.ReceiverIsSender) {
		return errors.Errorf("expected [%d] receiver is sender but got [%d]", len(m.Outputs), len(m.ReceiverIsSender))
	}
	return nil
}

// IsOutputAbsent returns true if the given output's metadata is absent
func (m *TransferMetadata) IsOutputAbsent(j int) bool {
	return len(m.TokenInfo[j]) == 0
}

// IsInputAbsent returns true if the given input's metadata is absent
func (m *TransferMetadata) IsInputAbsent(j int) bool {
	return m.Senders[j].IsNone()
}

func search(s []string, e string) int {
	for i, v := range s {
		if v == e {
			return i
		}
	}
	return -1
}
