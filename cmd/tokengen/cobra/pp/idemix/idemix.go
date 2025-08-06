/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"os"
	"path/filepath"

	"github.com/IBM/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// LoadIssuerPublicKey reads the issuer public key from the config file
func LoadIssuerPublicKey(idemixMSPDir string) (string, []byte, error) {
	// Load Idemix Issuer Public Key
	path := filepath.Join(idemixMSPDir, idemix.IdemixConfigDirMsp, idemix.IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := os.ReadFile(path)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed reading idemix issuer public key [%s]", path)
	}

	return path, ipkBytes, nil
}
