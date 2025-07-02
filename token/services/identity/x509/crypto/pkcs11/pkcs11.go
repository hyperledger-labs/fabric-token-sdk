//go:build pkcs11

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"os"

	"github.com/hyperledger/fabric-lib-go/bccsp"
	"github.com/hyperledger/fabric-lib-go/bccsp/pkcs11"
	"github.com/pkg/errors"
)

const (
	EnvPin       = "PKCS11_PIN"
	EnvLabel     = "PKCS11_LABEL"
	DefaultPin   = "98765432"
	DefaultLabel = "ForFSC"
)

type (
	PKCS11Opts   = pkcs11.PKCS11Opts
	KeyIDMapping = pkcs11.KeyIDMapping
)

// NewProvider returns a pkcs11 provider
func NewProvider(opts PKCS11Opts, ks bccsp.KeyStore, mapper func(ski []byte) []byte) (*pkcs11.Provider, error) {
	csp, err := pkcs11.New(opts, ks, pkcs11.WithKeyMapper(mapper))
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed initializing PKCS11 library with config [%+v]", opts)
	}
	return csp, nil
}

// FindPKCS11Lib attempts to find the PKCS11 library based on the given configuration
func FindPKCS11Lib() (lib, pin, label string, err error) {
	if lib = os.Getenv("PKCS11_LIB"); lib == "" {
		possibilities := []string{
			"/usr/lib/softhsm/libsofthsm2.so",                  // Debian
			"/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so", // Ubuntu
			"/usr/local/lib/softhsm/libsofthsm2.so",
			"/usr/lib/libacsp-pkcs11.so",
		}
		for _, path := range possibilities {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				lib = path
				break
			}
		}
	}
	if len(lib) == 0 {
		err = errors.New("cannot find PKCS11 lib")
	}
	if pin = os.Getenv(EnvPin); pin == "" {
		pin = DefaultPin
	}
	if label = os.Getenv(EnvLabel); label == "" {
		label = DefaultLabel
	}

	return
}
