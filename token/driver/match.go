/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"bytes"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Match validates the structural consistency between this metadata and the provided issue action.
// It checks input/output counts, extra signers, and the issuer identity.
func (i *IssueMetadata) Match(action IssueAction) error {
	if len(i.Inputs) != action.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(i.Inputs), action.NumInputs())
	}
	if len(i.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(i.Outputs), action.NumOutputs())
	}
	extraSigners := action.ExtraSigners()
	if len(i.ExtraSigners) != len(extraSigners) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(extraSigners), len(i.ExtraSigners))
	}
	for idx, signer := range extraSigners {
		if !slices.ContainsFunc(i.ExtraSigners, signer.Equal) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", signer, i.ExtraSigners[idx])
		}
	}
	if !i.Issuer.Identity.Equal(action.GetIssuer()) {
		return errors.Errorf("expected issuer [%s] but got [%s]", i.Issuer.Identity, action.GetIssuer())
	}

	return nil
}

// Match validates the structural consistency between this metadata and the provided transfer action.
// It checks input/output counts, extra signers, and the issuer identity.
func (t *TransferMetadata) Match(action TransferAction) error {
	if len(t.Inputs) != action.NumInputs() {
		return errors.Errorf("expected [%d] inputs but got [%d]", len(t.Inputs), action.NumInputs())
	}
	if len(t.Outputs) != action.NumOutputs() {
		return errors.Errorf("expected [%d] outputs but got [%d]", len(t.Outputs), action.NumOutputs())
	}
	extraSigners := action.ExtraSigners()
	if len(t.ExtraSigners) != len(extraSigners) {
		return errors.Errorf("expected [%d] extra signers but got [%d]", len(t.ExtraSigners), len(extraSigners))
	}
	for idx, signer := range extraSigners {
		if !signer.Equal(t.ExtraSigners[idx]) {
			return errors.Errorf("expected extra signer [%s] but got [%s]", t.ExtraSigners[idx], signer)
		}
	}
	if !t.Issuer.Equal(action.GetIssuer()) {
		return errors.Errorf("expected issuer [%s] but got [%s]", t.Issuer, action.GetIssuer().Bytes())
	}

	return nil
}

// MatchInputs validates that the serialized action inputs match the serialized ledger tokens.
// Nil entries in serializedActionInputs are skipped (e.g., upgrade-witness inputs).
func (t *TransferMetadata) MatchInputs(serializedActionInputs [][]byte, serializedLedgerTokens [][]byte) error {
	if len(serializedActionInputs) != len(serializedLedgerTokens) {
		return errors.Errorf("action has [%d] inputs but [%d] tokens provided", len(serializedActionInputs), len(serializedLedgerTokens))
	}
	for i, actionInput := range serializedActionInputs {
		if actionInput == nil {
			continue
		}
		if !bytes.Equal(actionInput, serializedLedgerTokens[i]) {
			return errors.Errorf("input token at index [%d]: does not match the transfer action", i)
		}
	}

	return nil
}

// ValidateReceivers checks that this output has at least one non-nil receiver declared.
func (t *TransferOutputMetadata) ValidateReceivers() error {
	if len(t.Receivers) == 0 {
		return errors.New("has no receivers")
	}
	for j, receiver := range t.Receivers {
		if receiver == nil {
			return errors.Errorf("receiver at index [%d] is nil", j)
		}
	}

	return nil
}
