/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	msp3 "github.com/hyperledger/fabric/msp"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"
	"github.com/pkg/errors"
)

const (
	signcerts = "signcerts"
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

	// read certificate from entries[0]/signcerts
	signcertDir := filepath.Join(entries[0], signcerts)
	content, err := GetPemMaterialFromDir(signcertDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load certificates from %s", signcertDir)
	}
	if len(content) == 0 {
		return nil, errors.Errorf("no certificates found in %s", signcertDir)
	}
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create x509 provider for [%s]", entry)
	}

	id, err := msp3.NewSerializedIdentity(mspID, content[0])
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create x509 identity for [%s]", entry)
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

func ReadFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %s", file)
	}

	return fileCont, nil
}

func ReadPemFile(file string) ([]byte, error) {
	bytes, err := ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading from file %s failed", file)
	}

	b, _ := pem.Decode(bytes)
	if b == nil { // TODO: also check that the type is what we expect (cert vs key..)
		return nil, errors.Errorf("no pem content for file %s", file)
	}

	return bytes, nil
}

func GetPemMaterialFromDir(dir string) ([][]byte, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, err
	}
	content := make([][]byte, 0)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read directory %s", dir)
	}
	for _, f := range files {
		fullName := filepath.Join(dir, f.Name())
		f, err := os.Stat(fullName)
		if err != nil {
			continue
		}
		if f.IsDir() {
			continue
		}
		item, err := ReadPemFile(fullName)
		if err != nil {
			continue
		}
		content = append(content, item)
	}
	return content, nil
}
