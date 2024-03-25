/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"encoding/hex"
	"path/filepath"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/pkg/errors"
)

// GetBCCSPFromConf returns a BCCSP instance and its relative key store from the passed configuration.
// If no configuration is passed, the default one is used, namely the `SW` provider.
func GetBCCSPFromConf(dir string, keyStorePath string, conf *BCCSP) (bccsp.BCCSP, bccsp.KeyStore, error) {
	if len(keyStorePath) == 0 {
		keyStorePath = filepath.Join(dir, "keystore")
	}
	if conf == nil {
		return GetSWBCCSP(keyStorePath)
	}

	switch conf.Default {
	case "SW":
		return GetSWBCCSP(keyStorePath)
	case "PKCS11":
		return GetPKCS11BCCSP(conf)
	default:
		return nil, nil, errors.Errorf("invalid BCCSP.Default.%s", conf.Default)
	}
}

// GetPKCS11BCCSP returns a new instance of the HSM-based BCCSP
func GetPKCS11BCCSP(conf *BCCSP) (bccsp.BCCSP, bccsp.KeyStore, error) {
	if conf.PKCS11 == nil {
		return nil, nil, errors.New("invalid BCCSP.PKCS11. missing configuration")
	}

	p11Opts := *conf.PKCS11
	ks := sw.NewDummyKeyStore()
	mapper := skiMapper(p11Opts)
	csp, err := pkcs11.New(*ToPKCS11OptsOpts(&p11Opts), ks, pkcs11.WithKeyMapper(mapper))
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "Failed initializing PKCS11 library with config [%+v]", p11Opts)
	}
	return csp, ks, nil
}

func skiMapper(p11Opts PKCS11) func([]byte) []byte {
	keyMap := map[string]string{}
	for _, k := range p11Opts.KeyIDs {
		keyMap[k.SKI] = k.ID
	}

	return func(ski []byte) []byte {
		keyID := hex.EncodeToString(ski)
		if id, ok := keyMap[keyID]; ok {
			return []byte(id)
		}
		if p11Opts.AltID != "" {
			return []byte(p11Opts.AltID)
		}
		return ski
	}
}

// GetSWBCCSP returns a new instance of the software-based BCCSP
func GetSWBCCSP(dir string) (bccsp.BCCSP, bccsp.KeyStore, error) {
	ks, err := sw.NewFileBasedKeyStore(nil, dir, true)
	if err != nil {
		return nil, nil, err
	}
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(ks)
	if err != nil {
		return nil, nil, err
	}
	return cryptoProvider, ks, nil
}
