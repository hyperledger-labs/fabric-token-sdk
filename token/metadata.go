/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TMS interface {
	// DeserializeToken returns the token and its issuer (if any).
	DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token.Token, view.Identity, error)
	// GetEnrollmentID extracts the enrollment id from the passed audit information
	GetEnrollmentID(bytes []byte) (string, error)
}

// Metadata contains the metadata of a Token Request
type Metadata struct {
	TMS                  TMS
	TokenRequestMetadata *driver.TokenRequestMetadata
}

// GetToken unmarshals the given bytes to extract the token and its issuer (if any).
func (m *Metadata) GetToken(raw []byte) (*token.Token, view.Identity, []byte, error) {
	tokenInfoRaw := m.TokenRequestMetadata.GetTokenInfo(raw)
	if len(tokenInfoRaw) == 0 {
		logger.Debugf("metadata for [%s] not found", hash.Hashable(raw).String())
		return nil, nil, nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(raw).String())
	}
	tok, id, err := m.TMS.DeserializeToken(raw, tokenInfoRaw)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed getting token in the clear")
	}
	return tok, id, tokenInfoRaw, nil
}

// SpentTokenID returns the token IDs of the tokens that were spent by the Token Request this metadata is associated with.
func (m *Metadata) SpentTokenID() []*token.ID {
	var res []*token.ID
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		res = append(res, transfer.TokenIDs...)
	}
	return res
}

// FilterBy returns a new Metadata containing only the metadata that matches the given enrollment IDs.
// For Issue actions, for each issue:
// - The sender;
// - The returned metadata will contain only the outputs whose owner has the given enrollment IDs.
// For Transfer actions, for each action:
// - The list of token IDs will be empty;
// - The returned metadata will contain only the outputs whose owner has the given enrollment IDs;
// - The senders are included if and only if there is at least one output whose owner has the given enrollment IDs.
// Application metadata is always included
func (m *Metadata) FilterBy(eIDs ...string) (*Metadata, error) {
	res := &Metadata{
		TMS:                  m.TMS,
		TokenRequestMetadata: &driver.TokenRequestMetadata{},
	}

	// filter issues
	for _, issue := range m.TokenRequestMetadata.Issues {
		issueRes := driver.IssueMetadata{
			Issuer: issue.Issuer,
		}

		for i, auditInfo := range issue.ReceiversAuditInfos {
			// If the receiver has the given enrollment ID, add it
			recipientEID, err := m.TMS.GetEnrollmentID(auditInfo)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting enrollment ID")
			}
			var Outputs []byte
			var TokenInfo []byte
			var Receivers view.Identity
			var ReceiverAuditInfos []byte

			if search(eIDs, recipientEID) != -1 {
				Outputs = issue.Outputs[i]
				TokenInfo = issue.TokenInfo[i]
				Receivers = issue.Receivers[i]
				ReceiverAuditInfos = issue.ReceiversAuditInfos[i]
			} else {
				logger.Debugf("skipping issue for [%s]", recipientEID)
			}

			issueRes.Outputs = append(issueRes.Outputs, Outputs)
			issueRes.TokenInfo = append(issueRes.TokenInfo, TokenInfo)
			issueRes.Receivers = append(issueRes.Receivers, Receivers)
			issueRes.ReceiversAuditInfos = append(issueRes.ReceiversAuditInfos, ReceiverAuditInfos)
		}

		res.TokenRequestMetadata.Issues = append(res.TokenRequestMetadata.Issues, issueRes)
	}

	// filter transfers
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		transferRes := driver.TransferMetadata{
			ExtraSigners: transfer.ExtraSigners,
		}

		// Filter outputs
		// if the receiver has the given enrollment ID, add it. Otherwise, add empty entries
		skip := true
		for i, auditInfo := range transfer.ReceiverAuditInfos {
			recipientEID, err := m.TMS.GetEnrollmentID(auditInfo)
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
				TokenInfo = transfer.OutputsMetadata[i]
				Receivers = transfer.Receivers[i]
				ReceiverIsSender = transfer.ReceiverIsSender[i]
				ReceiverAuditInfos = transfer.ReceiverAuditInfos[i]
				skip = false
			} else {
				logger.Debugf("skipping transfer for [%s]", recipientEID)
			}

			transferRes.Outputs = append(transferRes.Outputs, Outputs)
			transferRes.Receivers = append(transferRes.Receivers, Receivers)
			transferRes.ReceiverIsSender = append(transferRes.ReceiverIsSender, ReceiverIsSender)
			transferRes.ReceiverAuditInfos = append(transferRes.ReceiverAuditInfos, ReceiverAuditInfos)
			transferRes.OutputsMetadata = append(transferRes.OutputsMetadata, TokenInfo)
		}

		// if skip = true, it means that this transfer does not contain any output for the given enrollment IDs.
		// Therefore, no metadata should be given to the passed enrollment IDs.
		// if skip = false, it means that this transfer contains at least one output for the given enrollment IDs.
		// Append the senders to the transfer metadata.
		if !skip {
			for i, sender := range transfer.Senders {
				transferRes.Senders = append(transferRes.Senders, sender)
				transferRes.SenderAuditInfos = append(transferRes.SenderAuditInfos, transfer.SenderAuditInfos[i])
			}
		}

		logger.Debugf("keeping transfer with [%d] out of [%d] outputs", len(transferRes.Outputs), len(transfer.Outputs))
		res.TokenRequestMetadata.Transfers = append(res.TokenRequestMetadata.Transfers, transferRes)
	}

	// application
	res.TokenRequestMetadata.Application = m.TokenRequestMetadata.Application

	logger.Debugf("filtered metadata for [% x] from [%d:%d] to [%d:%d]",
		eIDs,
		len(m.TokenRequestMetadata.Issues), len(m.TokenRequestMetadata.Transfers),
		len(res.TokenRequestMetadata.Issues), len(res.TokenRequestMetadata.Transfers))

	return res, nil
}

// Issue returns the i-th issue metadata, if present
func (m *Metadata) Issue(i int) (*IssueMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Issues) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Issues))
	}
	return &IssueMetadata{IssueMetadata: &m.TokenRequestMetadata.Issues[i]}, nil
}

// Transfer returns the i-th transfer metadata, if present
func (m *Metadata) Transfer(i int) (*TransferMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Transfers) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Transfers))
	}
	return &TransferMetadata{TransferMetadata: &m.TokenRequestMetadata.Transfers[i]}, nil
}

// IssueMetadata contains the metadata of an issue action
type IssueMetadata struct {
	*driver.IssueMetadata
}

// Match returns true if the given action matches this metadata
func (m *IssueMetadata) Match(action *IssueAction) error {
	if action == nil {
		return errors.New("can't match issue metadata to issue action: nil issue action")
	}
	if len(m.Outputs) != 1 {
		return errors.Errorf("expected one output, got [%d]", len(m.Outputs))
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
	*driver.TransferMetadata
}

// Match returns true if the given action matches this metadata
func (m *TransferMetadata) Match(action *TransferAction) error {
	if action == nil {
		return errors.New("can't match transfer metadata to transfer action: nil issue action")
	}
	if len(m.TokenIDs) != 0 && len(m.Senders) != len(m.TokenIDs) {
		return errors.Errorf("expected [%d] token IDs and senders but got [%d]", len(m.TokenIDs), len(m.Senders))
	}
	if len(m.Senders) != len(m.SenderAuditInfos) {
		return errors.Errorf("expected [%d] senders and sender audit infos but got [%d]", len(m.Senders), len(m.SenderAuditInfos))
	}
	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}
	if len(m.Outputs) != len(m.OutputsMetadata) {
		return errors.Errorf("expected [%d] token info but got [%d]", len(m.Outputs), len(m.OutputsMetadata))
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
	if j >= len(m.OutputsMetadata) {
		return true
	}
	return len(m.OutputsMetadata[j]) == 0
}

// IsInputAbsent returns true if the given input's metadata is absent
func (m *TransferMetadata) IsInputAbsent(j int) bool {
	if j >= len(m.Senders) {
		return true
	}
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
