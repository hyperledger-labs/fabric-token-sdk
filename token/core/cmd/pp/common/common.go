/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"
	"github.com/pkg/errors"
)

// PP defines an interface shared by all public parameters
type PP interface {
	// AddAuditor adds an auditor to the public parameters
	AddAuditor(raw view.Identity)
	// AddIssuer adds an issuer to the public parameters
	AddIssuer(raw view.Identity)
}

// GetMSPIdentity returns the MSP identity from the passed entry formatted as <MSPConfigPath>:<MSPID>.
// If mspID is not empty, it will be used instead of the MSPID in the entry.
func GetMSPIdentity(entry string, mspID string) (view.Identity, error) {
	entries := strings.Split(entry, ":")
	if len(mspID) == 0 {
		if len(entries) != 2 {
			return nil, errors.Errorf("invalid input [%s], expected <MSPConfigPath>:<MSPID>", entry)
		}
		mspID = entries[1]
	} else {
		if len(entries) <= 0 || len(entries) > 2 {
			return nil, errors.Errorf("invalid input [%s], expected <MSPConfigPath>:<MSPID> or <MSPConfigPath>", entry)
		}
	}
	provider, err := x509.NewProvider(entries[0], mspID, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create x509 provider for [%s]", entry)
	}
	id, _, err := provider.Identity(nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get identity [%s]", entry)
	}
	return id, nil
}

// SetupIssuersAndAuditors sets up the issuers and auditors for the given public parameters
func SetupIssuersAndAuditors(pp PP, Auditors, Issuers []string) error {
	// Auditors
	for _, auditor := range Auditors {
		id, err := GetMSPIdentity(auditor, msp.AuditorMSPID)
		if err != nil {
			return errors.WithMessagef(err, "failed to get auditor identity [%s]", auditor)
		}
		pp.AddAuditor(id)
	}
	// Issuers
	for _, issuer := range Issuers {
		id, err := GetMSPIdentity(issuer, msp.IssuerMSPID)
		if err != nil {
			return errors.WithMessagef(err, "failed to get issuer identity [%s]", issuer)
		}
		pp.AddIssuer(id)
	}
	return nil
}
