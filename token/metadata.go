/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Metadata contains the metadata of a Token Request.
// This metadata is used to unscramble the content of the actions in the Token Request.
// It includes information about issuers, owners, and extra signers.
type Metadata struct {
	// TokenService is the service used to handle tokens.
	TokenService driver.TokensService
	// WalletService is the service used to handle wallets and identities.
	WalletService driver.WalletService
	// TokenRequestMetadata contains the metadata of the Token Request as defined by the driver.
	TokenRequestMetadata *driver.TokenRequestMetadata
	// Logger is the logger used by this struct.
	Logger logging.Logger
}

// SpentTokenID returns the token IDs of the tokens that were spent by the Token Request this metadata is associated with.
// It iterates over all issue and transfer actions and collects the IDs of the input tokens.
func (m *Metadata) SpentTokenID() []*token.ID {
	var res []*token.ID
	// Collect token IDs from issue actions.
	// Note: issue actions usually don't have inputs, but some drivers might use them (e.g., for token upgrades).
	for _, issue := range m.TokenRequestMetadata.Issues {
		for _, input := range issue.Inputs {
			res = append(res, input.TokenID)
		}
	}
	// Collect token IDs from transfer actions.
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		for _, input := range transfer.Inputs {
			res = append(res, input.TokenID)
		}
	}

	return res
}

// FilterBy returns a new Metadata containing only the metadata that matches the given enrollment IDs.
// This is used to share with a party only the metadata they are entitled to see.
//
// For Issue actions:
// - The issuer information is always included.
// - Only metadata for outputs owned by the given enrollment IDs is included.
//
// For Transfer actions:
// - Only metadata for outputs owned by the given enrollment IDs is included.
// - Sender information is included if and only if there is at least one output owned by the given enrollment IDs.
//
// Application metadata is always included.
func (m *Metadata) FilterBy(ctx context.Context, eIDs ...string) (*Metadata, error) {
	if len(eIDs) == 0 {
		return m, nil
	}

	eIDSet := collections.NewSet(eIDs...)

	// Filter issue metadata.
	issues, err := m.filterIssues(ctx, m.TokenRequestMetadata.Issues, eIDSet)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed filtering issues")
	}
	// Filter transfer metadata.
	transfers, err := m.filterTransfers(ctx, m.TokenRequestMetadata.Transfers, eIDSet)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed filtering transfers")
	}
	clone := &Metadata{
		TokenService:  m.TokenService,
		WalletService: m.WalletService,
		TokenRequestMetadata: &driver.TokenRequestMetadata{
			Issues:      issues,
			Transfers:   transfers,
			Application: m.TokenRequestMetadata.Application,
		},
		Logger: m.Logger,
	}

	m.Logger.Debugf("filtered metadata for [% x] from [%d:%d] to [%d:%d]",
		eIDs,
		len(m.TokenRequestMetadata.Issues), len(m.TokenRequestMetadata.Transfers),
		len(clone.TokenRequestMetadata.Issues), len(clone.TokenRequestMetadata.Transfers))

	return clone, nil
}

// filterIssues filters the issue metadata based on the provided enrollment IDs.
func (m *Metadata) filterIssues(ctx context.Context, issues []*driver.IssueMetadata, eIDSet collections.Set[string]) ([]*driver.IssueMetadata, error) {
	cloned := make([]*driver.IssueMetadata, 0, len(issues))
	for _, issue := range m.TokenRequestMetadata.Issues {
		clone := &driver.IssueMetadata{
			Issuer:       issue.Issuer,
			Inputs:       issue.Inputs,
			Outputs:      nil,
			ExtraSigners: issue.ExtraSigners,
		}

		counter := 0
		for _, output := range issue.Outputs {
			// Check if any of the receivers of the output matches the enrollment IDs.
			if found, err := m.contains(ctx, output.Receivers, eIDSet); err != nil {
				return nil, errors.WithMessagef(err, "failed checking receivers")
			} else if found {
				// If matched, include the full output metadata.
				clone.Outputs = append(clone.Outputs, output)
				counter++
			} else {
				// If not matched, include a nil entry to preserve the indexing.
				clone.Outputs = append(clone.Outputs, nil)
			}
		}

		m.Logger.Debugf("keeping issue with [%d] out of [%d] outputs", counter, len(issue.Outputs))
		cloned = append(cloned, clone)
	}

	return cloned, nil
}

// filterTransfers filters the transfer metadata based on the provided enrollment IDs.
func (m *Metadata) filterTransfers(ctx context.Context, issues []*driver.TransferMetadata, eIDSet collections.Set[string]) ([]*driver.TransferMetadata, error) {
	cloned := make([]*driver.TransferMetadata, 0, len(issues))
	for _, transfer := range m.TokenRequestMetadata.Transfers {
		clone := &driver.TransferMetadata{
			Inputs:       nil,
			Outputs:      nil,
			ExtraSigners: transfer.ExtraSigners,
		}

		// Filter outputs: if the receiver has the given enrollment ID, add it. Otherwise, add empty entries.
		counter := 0
		for _, output := range transfer.Outputs {
			if found, err := m.contains(ctx, output.Receivers, eIDSet); err != nil {
				return nil, errors.WithMessagef(err, "failed checking receivers")
			} else if found {
				clone.Outputs = append(clone.Outputs, output)
				counter++
			} else {
				clone.Outputs = append(clone.Outputs, nil)
			}
		}

		// Prepare empty input metadata entries.
		for range transfer.Inputs {
			clone.Inputs = append(clone.Inputs, &driver.TransferInputMetadata{})
		}
		// If at least one output matched, include the sender information for all inputs.
		if counter > 0 {
			for i, input := range transfer.Inputs {
				clone.Inputs[i].Senders = input.Senders
			}
		}

		m.Logger.Debugf("keeping transfer with [%d] out of [%d] outputs", counter, len(transfer.Outputs))
		cloned = append(cloned, clone)
	}

	return cloned, nil
}

// contains checks if any of the given auditable identities matches the provided enrollment IDs.
func (m *Metadata) contains(ctx context.Context, receivers []*driver.AuditableIdentity, eIDSet collections.Set[string]) (bool, error) {
	for _, receiver := range receivers {
		// Resolve the enrollment ID of the receiver.
		recipientEID, err := m.WalletService.GetEnrollmentID(ctx, receiver.Identity, receiver.AuditInfo)
		if err != nil {
			return false, errors.Wrap(err, "failed getting enrollment ID")
		}
		// Check if the enrollment ID is in the set.
		if eIDSet.Contains(recipientEID) {
			logger.Debugf("eid [%s] found in list [%v]", recipientEID, eIDSet)

			return true, nil
		} else {
			logger.Debugf("eid [%s] not found in list [%v]", recipientEID, eIDSet)
		}
	}

	return false, nil
}

// Issue returns the i-th issue metadata, if present.
func (m *Metadata) Issue(i int) (*IssueMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Issues) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Issues))
	}

	return &IssueMetadata{IssueMetadata: m.TokenRequestMetadata.Issues[i]}, nil
}

// Transfer returns the i-th transfer metadata, if present.
func (m *Metadata) Transfer(i int) (*TransferMetadata, error) {
	if i >= len(m.TokenRequestMetadata.Transfers) {
		return nil, errors.Errorf("index [%d] out of range [0:%d]", i, len(m.TokenRequestMetadata.Transfers))
	}

	return &TransferMetadata{TransferMetadata: m.TokenRequestMetadata.Transfers[i]}, nil
}

// IssueMetadata contains the metadata of an issue action.
type IssueMetadata struct {
	*driver.IssueMetadata
}

// Match returns true if the given action matches this metadata.
// It performs a deep check of inputs, outputs, extra signers, and the issuer identity.
func (m *IssueMetadata) Match(action *IssueAction) error {
	if action == nil {
		return errors.New("can't match issue metadata to issue action: nil issue action")
	}

	// Validate the action's structure.
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "failed validating issue action")
	}

	// Check that the number of inputs matches.
	if len(m.Inputs) != action.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(m.Inputs), action.NumInputs())
	}

	// Check that the number of outputs matches.
	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}

	// Check that the extra signers are the same.
	extraSigners := action.a.ExtraSigners()
	if len(m.ExtraSigners) != len(extraSigners) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(extraSigners), len(m.ExtraSigners))
	}
	for i, signer := range extraSigners {
		if !slices.ContainsFunc(m.ExtraSigners, signer.Equal) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", signer, m.ExtraSigners[i])
		}
	}

	// Check that the issuer identity matches.
	if !m.Issuer.Identity.Equal(action.GetIssuer()) {
		return errors.Errorf("expected issuer [%s] but got [%s]", m.Issuer.Identity, action.GetIssuer())
	}

	return nil
}

// IsOutputAbsent returns true if the j-th output's metadata is absent (e.g., filtered out).
func (m *IssueMetadata) IsOutputAbsent(j int) bool {
	if j < 0 || j >= len(m.Outputs) {
		return true
	}

	return m.Outputs[j] == nil
}

// TransferMetadata contains the metadata of a transfer action.
type TransferMetadata struct {
	*driver.TransferMetadata
}

// Match returns true if the given action matches this metadata.
// It performs a deep check of inputs, outputs, extra signers, and the issuer identity (if present).
func (m *TransferMetadata) Match(action *TransferAction) error {
	if action == nil {
		return errors.New("can't match transfer metadata to transfer action: nil issue action")
	}

	// Validate the action's structure.
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "failed validating issue action")
	}

	// Check that the number of inputs matches.
	if len(m.Inputs) != action.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(m.Inputs), action.NumInputs())
	}

	// Check that the number of outputs matches.
	if len(m.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(m.Outputs), action.NumOutputs())
	}

	// Check that the extra signers are the same.
	extraSigners := action.ExtraSigners()
	if len(m.ExtraSigners) != len(extraSigners) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(m.ExtraSigners), len(extraSigners))
	}
	for i, signer := range extraSigners {
		if !signer.Equal(m.ExtraSigners[i]) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", m.ExtraSigners[i], signer)
		}
	}

	// Check that the issuer identity matches, if present in the metadata.
	if !m.Issuer.Equal(action.GetIssuer()) {
		return errors.Errorf("expected issuer [%s] but got [%s]", m.Issuer, action.GetIssuer().Bytes())
	}

	return nil
}

// IsOutputAbsent returns true if the j-th output's metadata is absent (e.g., filtered out).
func (m *TransferMetadata) IsOutputAbsent(j int) bool {
	if j < 0 || j >= len(m.Outputs) {
		return true
	}

	return m.Outputs[j] == nil
}

// IsInputAbsent returns true if the j-th input's metadata is absent (e.g., filtered out).
func (m *TransferMetadata) IsInputAbsent(j int) bool {
	if j < 0 || j >= len(m.Inputs) {
		return true
	}

	return m.Inputs[j] == nil || len(m.Inputs[j].Senders) == 0
}
