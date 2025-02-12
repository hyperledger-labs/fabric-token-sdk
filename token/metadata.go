/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// Metadata contains the metadata of a Token Request
type Metadata struct {
	TokenService         driver.TokensService
	WalletService        driver.WalletService
	TokenRequestMetadata *driver.TokenRequestMetadata
	Logger               logging.Logger
}

// SpentTokenID returns the token IDs of the tokens that were spent by the Token Request this metadata is associated with.
func (m *Metadata) SpentTokenID() []*token.ID {
	var res []*token.ID
	for _, issue := range m.TokenRequestMetadata.Issues {
		for _, input := range issue.Inputs {
			res = append(res, input.TokenID)
		}
	}
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		for _, input := range transfer.Inputs {
			res = append(res, input.TokenID)
		}
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
	if len(eIDs) == 0 {
		return m, nil
	}

	clonedMetadata := &Metadata{
		TokenService:         m.TokenService,
		WalletService:        m.WalletService,
		TokenRequestMetadata: &driver.TokenRequestMetadata{},
		Logger:               m.Logger,
	}
	eIDSet := collections.NewSet(eIDs...)
	// filter issues
	for _, issue := range m.TokenRequestMetadata.Issues {
		clone := &driver.IssueMetadata{
			Issuer:       issue.Issuer,
			Inputs:       issue.Inputs,
			Outputs:      nil,
			ExtraSigners: issue.ExtraSigners,
		}

		counter := 0
		for _, output := range issue.Outputs {
			found := false
			for _, receiver := range output.Receivers {
				// If the receiver has the given enrollment ID, add it
				recipientEID, err := m.WalletService.GetEnrollmentID(receiver.Identity, receiver.AuditInfo)
				if err != nil {
					return nil, errors.Wrap(err, "failed getting enrollment ID")
				}
				if eIDSet.Contains(recipientEID) {
					found = true
					break
				}
			}
			if found {
				clone.Outputs = append(clone.Outputs, output)
				counter++
			} else {
				clone.Outputs = append(clone.Outputs, nil)
			}
		}

		m.Logger.Debugf("keeping issue with [%d] out of [%d] outputs", counter, len(issue.Outputs))
		clonedMetadata.TokenRequestMetadata.Issues = append(clonedMetadata.TokenRequestMetadata.Issues, clone)
	}

	// filter transfers
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		clone := &driver.TransferMetadata{
			Inputs:       nil,
			Outputs:      nil,
			ExtraSigners: transfer.ExtraSigners,
		}

		// Filter outputs
		// if the receiver has the given enrollment ID, add it. Otherwise, add empty entries
		skip := true
		counter := 0
		for _, output := range transfer.Outputs {
			found := false
			for _, receiver := range output.Receivers {
				// If the receiver has the given enrollment ID, add it
				recipientEID, err := m.WalletService.GetEnrollmentID(receiver.Identity, receiver.AuditInfo)
				if err != nil {
					return nil, errors.Wrap(err, "failed getting enrollment ID")
				}
				if eIDSet.Contains(recipientEID) {
					skip = false
					found = true
					break
				}
			}
			if found {
				clone.Outputs = append(clone.Outputs, output)
				counter++
			} else {
				clone.Outputs = append(clone.Outputs, nil)
			}
		}

		// if skip = true, it means that this transfer does not contain any output for the given enrollment IDs.
		// Therefore, no metadata should be given to the passed enrollment IDs.
		// if skip = false, it means that this transfer contains at least one output for the given enrollment IDs.
		// Append the senders to the transfer metadata.
		for range transfer.Inputs {
			clone.Inputs = append(clone.Inputs, &driver.TransferInputMetadata{
				TokenID: nil,
				Senders: nil,
			})
		}
		if !skip {
			for i, input := range transfer.Inputs {
				clone.Inputs[i].Senders = input.Senders
			}
		}

		m.Logger.Debugf("keeping transfer with [%d] out of [%d] outputs", counter, len(transfer.Outputs))
		clonedMetadata.TokenRequestMetadata.Transfers = append(clonedMetadata.TokenRequestMetadata.Transfers, clone)
	}

	// application
	clonedMetadata.TokenRequestMetadata.Application = m.TokenRequestMetadata.Application

	// TODO: update this log
	m.Logger.Debugf("filtered metadata for [% x] from [%d:%d] to [%d:%d]",
		eIDs,
		len(m.TokenRequestMetadata.Issues), len(m.TokenRequestMetadata.Transfers),
		len(clonedMetadata.TokenRequestMetadata.Issues), len(clonedMetadata.TokenRequestMetadata.Transfers))

	return clonedMetadata, nil
}

// Issue returns the i-th issue metadata, if present
func (m *Metadata) Issue(i int) (*IssueMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Issues) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Issues))
	}
	return &IssueMetadata{IssueMetadata: m.TokenRequestMetadata.Issues[i]}, nil
}

// Transfer returns the i-th transfer metadata, if present
func (m *Metadata) Transfer(i int) (*TransferMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Transfers) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Transfers))
	}
	return &TransferMetadata{TransferMetadata: m.TokenRequestMetadata.Transfers[i]}, nil
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

	// validate action
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "failed validating issue action")
	}

	// check inputs
	if len(m.Inputs) != action.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(m.Inputs), action.NumInputs())
	}

	// check outputs
	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}

	// extra signer
	extraSigner := action.a.ExtraSigners()
	if len(m.ExtraSigners) != len(extraSigner) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(extraSigner), len(m.ExtraSigners))
	}
	// check that the extra signers are the same
	for i, signer := range extraSigner {
		if !signer.Equal(m.ExtraSigners[i]) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", signer, m.ExtraSigners[i])
		}
	}
	return nil
}

// IsOutputAbsent returns true if the given output's metadata is absent
func (m *IssueMetadata) IsOutputAbsent(j int) bool {
	if j < 0 || j >= len(m.Outputs) {
		return true
	}
	return m.Outputs[j] == nil
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

	// validate action
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "failed validating issue action")
	}

	// inputs
	if len(m.Inputs) != action.a.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(m.Inputs), action.a.NumInputs())
	}

	// outputs
	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}

	// extra signer
	extraSigner := action.a.ExtraSigners()
	if len(m.ExtraSigners) != len(extraSigner) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(m.ExtraSigners), len(extraSigner))
	}
	// check that the extra signers are the same
	for i, signer := range extraSigner {
		if !signer.Equal(m.ExtraSigners[i]) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", m.ExtraSigners[i], signer)
		}
	}
	return nil
}

// IsOutputAbsent returns true if the given output's metadata is absent
func (m *TransferMetadata) IsOutputAbsent(j int) bool {
	if j >= len(m.Outputs) {
		return true
	}
	return m.Outputs[j] == nil
}

// IsInputAbsent returns true if the given input's metadata is absent
func (m *TransferMetadata) IsInputAbsent(j int) bool {
	if j >= len(m.Inputs) {
		return true
	}
	return m.Inputs[j] == nil || len(m.Inputs[j].Senders) == 0
}
