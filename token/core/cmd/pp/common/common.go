package common

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

// PP defines an interface shared by all public parameters
type PP interface {
	// AddAuditor adds an auditor to the public parameters
	AddAuditor(raw view.Identity)
	// AddIssuer adds an issuer to the public parameters
	AddIssuer(raw view.Identity)
}

// GetMSPIdentity returns the MSP identity from the passed entry formatted as <MSPConfigPath>:<MSPID>
func GetMSPIdentity(entry string) (view.Identity, error) {
	entries := strings.Split(entry, ":")
	if len(entries) != 2 {
		return nil, errors.Errorf("invalid input [%s]", entry)
	}
	provider, err := x509.NewProvider(entries[0], entries[1], nil)
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
		id, err := GetMSPIdentity(auditor)
		if err != nil {
			return errors.WithMessagef(err, "failed to get auditor identity [%s]", auditor)
		}
		pp.AddAuditor(id)
	}
	// Issuers
	for _, issuer := range Issuers {
		id, err := GetMSPIdentity(issuer)
		if err != nil {
			return errors.WithMessagef(err, "failed to get issuer identity [%s]", issuer)
		}
		pp.AddIssuer(id)
	}
	return nil
}
