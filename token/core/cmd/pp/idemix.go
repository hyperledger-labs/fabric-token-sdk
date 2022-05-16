/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	IdemixConfigDirMsp              = "msp"
	IdemixConfigFileIssuerPublicKey = "IssuerPublicKey"
)

// LoadIdemixIssuerPublicKey reads the issuer public key from the config file
func LoadIdemixIssuerPublicKey(args *GeneratorArgs) (string, []byte, error) {
	// Load Idemix Issuer Public Key
	path := filepath.Join(args.IdemixMSPDir, IdemixConfigDirMsp, IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed reading idemix issuer public key")
	}

	return path, ipkBytes, nil
}
