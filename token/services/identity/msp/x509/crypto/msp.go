/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/signer"
	"github.com/pkg/errors"
)

type verifyingIdentity struct {
	bccsp               bccsp.BCCSP
	SignatureHashFamily string
	cert                *x509.Certificate
	Mspid               string
	pk                  bccsp.Key
}

func (f *verifyingIdentity) Serialize() ([]byte, error) {
	pb := &pem.Block{Bytes: f.cert.Raw, Type: "CERTIFICATE"}
	pemBytes := pem.EncodeToMemory(pb)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

	// We serialize identities by prepending the MSPID and appending the ASN.1 DER content of the cert
	sId := &SerializedIdentity{Mspid: f.Mspid, IdBytes: pemBytes}
	idBytes, err := proto.Marshal(sId)
	if err != nil {
		return nil, errors.WithMessage(err, "failed serializing identity")
	}

	return idBytes, nil
}

func (f *verifyingIdentity) Verify(message, sigma []byte) error {
	hashOpt, err := getHashOpt(f.SignatureHashFamily)
	if err != nil {
		return errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := f.bccsp.Hash(message, hashOpt)
	if err != nil {
		return errors.WithMessage(err, "failed computing digest")
	}

	valid, err := f.bccsp.Verify(f.pk, sigma, digest, nil)
	if err != nil {
		return errors.WithMessage(err, "could not determine the validity of the signature")
	} else if !valid {
		return errors.New("signature is invalid")
	}

	return nil
}

type fullIdentity struct {
	*verifyingIdentity
	signer crypto.Signer
}

func (f *fullIdentity) Sign(msg []byte) ([]byte, error) {
	// Compute Hash
	hashOpt, err := getHashOpt(f.SignatureHashFamily)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := f.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed computing digest")
	}

	// Sign
	return f.signer.Sign(rand.Reader, digest, nil)
}

func (f *fullIdentity) Serialize() ([]byte, error) {
	pb := &pem.Block{Bytes: f.cert.Raw, Type: "CERTIFICATE"}
	pemBytes := pem.EncodeToMemory(pb)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

	// We serialize identities by prepending the MSPID and appending the ASN.1 DER content of the cert
	sId := &SerializedIdentity{Mspid: f.Mspid, IdBytes: pemBytes}
	idBytes, err := proto.Marshal(sId)
	if err != nil {
		return nil, errors.WithMessage(err, "failed serializing identity")
	}

	return idBytes, nil
}

func (f *fullIdentity) Verify(message, sigma []byte) error {
	hashOpt, err := getHashOpt(f.SignatureHashFamily)
	if err != nil {
		return errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := f.bccsp.Hash(message, hashOpt)
	if err != nil {
		return errors.WithMessage(err, "failed computing digest")
	}

	valid, err := f.bccsp.Verify(f.pk, sigma, digest, nil)
	if err != nil {
		return errors.WithMessage(err, "could not determine the validity of the signature")
	} else if !valid {
		return errors.New("signature is invalid")
	}

	return nil
}

type IdentityFactory struct {
	bccsp               bccsp.BCCSP
	SignatureHashFamily string
}

func NewIdentityFactory(bccsp bccsp.BCCSP, signatureHashFamily string) *IdentityFactory {
	return &IdentityFactory{bccsp: bccsp, SignatureHashFamily: signatureHashFamily}
}

func (f *IdentityFactory) GetFullIdentity(sidInfo *SigningIdentityInfo) (*fullIdentity, error) {
	if sidInfo == nil {
		return nil, errors.New("getIdentityFromBytes error: nil sidInfo")
	}

	// Extract the public part of the identity
	idPub, pubKey, err := f.getIdentityFromConf(sidInfo.PublicSigner)
	if err != nil {
		return nil, err
	}

	// Find the matching private key in the BCCSP keystore
	privKey, err := f.bccsp.GetKey(pubKey.SKI())
	// Less Secure: Attempt to import Private Key from KeyInfo, if BCCSP was not able to find the key
	if err != nil {
		if sidInfo.PrivateSigner == nil || sidInfo.PrivateSigner.KeyMaterial == nil {
			return nil, errors.New("KeyMaterial not found in SigningIdentityInfo")
		}
		pemKey, _ := pem.Decode(sidInfo.PrivateSigner.KeyMaterial)
		if pemKey == nil {
			return nil, errors.Errorf("%s: wrong PEM encoding", sidInfo.PrivateSigner.KeyIdentifier)
		}
		privKey, err = f.bccsp.KeyImport(pemKey.Bytes, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: true})
		if err != nil {
			return nil, errors.WithMessage(err, "getIdentityFromBytes error: Failed to import EC private key")
		}
	}

	// get the peer signer
	identitySigner, err := signer.New(f.bccsp, privKey)
	if err != nil {
		return nil, errors.WithMessage(err, "getIdentityFromBytes error: Failed initializing bccspCryptoSigner")
	}

	return &fullIdentity{
		verifyingIdentity: &verifyingIdentity{
			bccsp:               f.bccsp,
			SignatureHashFamily: f.SignatureHashFamily,
			Mspid:               "",
			cert:                idPub,
			pk:                  pubKey,
		},
		signer: identitySigner,
	}, nil
}

func (f *IdentityFactory) GetIdentity(sidInfo *SigningIdentityInfo) (*verifyingIdentity, error) {
	if sidInfo == nil {
		return nil, errors.New("getIdentityFromBytes error: nil sidInfo")
	}

	// Extract the public part of the identity
	idPub, pubKey, err := f.getIdentityFromConf(sidInfo.PublicSigner)
	if err != nil {
		return nil, err
	}

	return &verifyingIdentity{
		bccsp:               f.bccsp,
		SignatureHashFamily: f.SignatureHashFamily,
		Mspid:               "",
		cert:                idPub,
		pk:                  pubKey,
	}, nil
}

func (f *IdentityFactory) getIdentityFromConf(idBytes []byte) (*x509.Certificate, bccsp.Key, error) {
	// get a cert
	cert, err := f.getCertFromPem(idBytes)
	if err != nil {
		return nil, nil, err
	}

	// get the public key in the right format
	certPubK, err := f.bccsp.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, nil, err
	}

	return cert, certPubK, nil
}

func (f *IdentityFactory) getCertFromPem(idBytes []byte) (*x509.Certificate, error) {
	if idBytes == nil {
		return nil, errors.New("getCertFromPem error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCertFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	var cert *x509.Certificate
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCertFromPem error: failed to parse x509 cert")
	}

	return cert, nil
}

func getHashOpt(hashFamily string) (bccsp.HashOpts, error) {
	switch hashFamily {
	case bccsp.SHA2:
		return bccsp.GetHashOpt(bccsp.SHA256)
	case bccsp.SHA3:
		return bccsp.GetHashOpt(bccsp.SHA3_256)
	}
	return nil, errors.Errorf("hash family not recognized [%s]", hashFamily)
}
