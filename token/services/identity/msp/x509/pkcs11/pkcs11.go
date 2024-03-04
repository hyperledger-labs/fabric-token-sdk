/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"os"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/bccsp/sw"
	pkcs11lib "github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

const (
	EnvPin       = "PKCS11_PIN"
	EnvLabel     = "PKCS11_LABEL"
	DefaultPin   = "98765432"
	DefaultLabel = "ForFSC"

	Provider = "PKCS11"
)

var logger = flogging.MustGetLogger("nwo.common.pkcs11")

// GeneratePrivateKey creates a private key in the HSM and returns its corresponding public key
func GeneratePrivateKey() (*ecdsa.PublicKey, error) {
	lib, pin, label, err := FindPKCS11Lib()
	if err != nil {
		return nil, err
	}
	csp, _, err := GetBCCSP(&config.BCCSP{
		Default: "PKCS11",
		PKCS11: &config.PKCS11{
			Security: 256,
			Hash:     "SHA2",
			Library:  lib,
			Pin:      pin,
			Label:    label,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "GeneratePrivateKey: Failed initializing PKCS11 library")
	}

	key, err := csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: false})
	if err != nil {
		return nil, errors.Wrap(err, "Failed generating ECDSA P256 key")
	}

	pub, err := key.PublicKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting public key")
	}

	raw, err := pub.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling public key")
	}

	pk, err := DERToPublicKey(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling der to public key")
	}

	return pk.(*ecdsa.PublicKey), nil
}

// GetBCCSP returns a new instance of the HSM-based BCCSP
func GetBCCSP(conf *config.BCCSP) (bccsp.BCCSP, bccsp.KeyStore, error) {
	if conf.PKCS11 == nil {
		return nil, nil, errors.New("invalid config.BCCSP.PKCS11. missing configuration")
	}

	if err := CheckToken(conf.PKCS11); err != nil {
		return nil, nil, errors.Wrap(err, "failed to check token")
	}

	p11Opts := *conf.PKCS11
	ks := sw.NewDummyKeyStore()
	mapper := skiMapper(p11Opts)
	csp, err := pkcs11.New(*ToPKCS11Opts(&p11Opts), ks, pkcs11.WithKeyMapper(mapper))
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "GetBCCSP: Failed initializing PKCS11 library with config [%+v]", p11Opts)
	}
	return csp, ks, nil
}

func ToPKCS11Opts(o *config.PKCS11) *pkcs11.PKCS11Opts {
	res := &pkcs11.PKCS11Opts{
		Security:       o.Security,
		Hash:           o.Hash,
		Library:        o.Library,
		Label:          o.Label,
		Pin:            o.Pin,
		SoftwareVerify: o.SoftwareVerify,
		Immutable:      o.Immutable,
		AltID:          o.AltID,
	}
	for _, d := range o.KeyIDs {
		res.KeyIDs = append(res.KeyIDs, pkcs11.KeyIDMapping{
			SKI: d.SKI,
			ID:  d.ID,
		})
	}
	return res
}

func skiMapper(p11Opts config.PKCS11) func([]byte) []byte {
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

// DERToPublicKey unmarshals a der to public key
func DERToPublicKey(raw []byte) (pub interface{}, err error) {
	if len(raw) == 0 {
		return nil, errors.New("Invalid DER. It must be different from nil.")
	}
	key, err := x509.ParsePKIXPublicKey(raw)
	return key, err
}

// CheckToken checks whether a token is available that matches the given label.
// If the token does not exist, it is created.
func CheckToken(opts *config.PKCS11) error {
	if opts.Library == "" {
		return errors.Errorf("pkcs11: library path not provided")
	}

	ctx := pkcs11lib.New(opts.Library)
	if ctx == nil {
		return errors.Errorf("pkcs11: instantiation failed for %s", opts.Library)
	}
	defer func() {
		if err := ctx.Finalize(); err != nil {
			logger.Errorf("failed context finalization [%s]", err)
		}
	}()
	if err := ctx.Initialize(); err != nil && !strings.Contains(err.Error(), "CKR_CRYPTOKI_ALREADY_INITIALIZED") {
		return errors.Errorf("pkcs11: initialization failed: %v", err)
	}

	slots, err := ctx.GetSlotList(false)
	if err != nil {
		return errors.Errorf("pkcs11: failed to get slot list: %v", err)
	}

	for _, s := range slots {
		info, err := ctx.GetTokenInfo(s)
		if err != nil {
			logger.Debugf("pkcs11: failed to get token info for slot %d: %v", s, err)
			continue
		}
		if info.Label == opts.Label {
			logger.Debugf("pkcs11: found token with label %s on slot %d \n", opts.Label, s)
			return nil
		}
	}

	logger.Debugf("pkcs11: no token with label %s found, create one with PIN %s", opts.Label, opts.Pin)

	// create token
	if err := ctx.InitToken(0, opts.Pin, opts.Label); err != nil {
		return errors.Errorf("pkcs11: failed to initialize token: %v", err)
	}

	slots, err = ctx.GetSlotList(true)
	if err != nil {
		return errors.Wrap(err, "pkcs11: get slot list")
	}
	var slot uint
	for _, s := range slots {
		info, err := ctx.GetTokenInfo(s)
		if err != nil || opts.Label != info.Label {
			continue
		}
		slot = s
		break
	}

	sess, err := ctx.OpenSession(slot, pkcs11lib.CKF_SERIAL_SESSION|pkcs11lib.CKF_RW_SESSION)
	if err != nil {
		return errors.Wrap(err, "pkcs11: open session")
	}

	defer func() {
		if err := ctx.CloseSession(sess); err != nil {
			logger.Errorf("failed closing session [%s]", err)
		}
	}()
	if err := ctx.Login(sess, pkcs11lib.CKU_USER, opts.Pin); err != nil {
		return errors.Wrap(err, "pkcs11: login")
	}
	if err := ctx.InitPIN(sess, opts.Pin); err != nil {
		return errors.Wrap(err, "pkcs11: init pin")
	}

	return nil
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
