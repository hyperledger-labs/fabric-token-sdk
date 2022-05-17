/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	ConfigDirMsp              = "msp"
	ConfigFileIssuerPublicKey = "IssuerPublicKey"
)

// LoadIssuerPublicKey reads the issuer public key from the config file
func LoadIssuerPublicKey(idemixMSPDir string) (string, []byte, error) {
	// Load Idemix Issuer Public Key
	path := filepath.Join(idemixMSPDir, ConfigDirMsp, ConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed reading idemix issuer public key [%s]", path)
	}

	return path, ipkBytes, nil
}
