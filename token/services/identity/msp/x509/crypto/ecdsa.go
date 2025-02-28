/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

var (
	// curveHalfOrders contains the precomputed curve group orders halved.
	// It is used to ensure that signature' S value is lower or equal to the
	// curve group order halved. We accept only low-S signatures.
	// They are precomputed for efficiency reasons.
	curveHalfOrders = map[elliptic.Curve]*big.Int{
		elliptic.P224(): new(big.Int).Rsh(elliptic.P224().Params().N, 1),
		elliptic.P256(): new(big.Int).Rsh(elliptic.P256().Params().N, 1),
		elliptic.P384(): new(big.Int).Rsh(elliptic.P384().Params().N, 1),
		elliptic.P521(): new(big.Int).Rsh(elliptic.P521().Params().N, 1),
	}
)

func IsECDSASignedCert(cert *x509.Certificate) bool {
	return cert.SignatureAlgorithm == x509.ECDSAWithSHA1 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA256 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA384 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA512
}

type ECDSASignature struct {
	R, S *big.Int
}

type ecdsaVerifier struct {
	pk *ecdsa.PublicKey
}

func NewECDSAVerifier(pk *ecdsa.PublicKey) *ecdsaVerifier {
	return &ecdsaVerifier{pk: pk}
}

func (d *ecdsaVerifier) Verify(message, sigma []byte) error {
	signature := &ECDSASignature{}
	_, err := asn1.Unmarshal(sigma, signature)
	if err != nil {
		return err
	}

	hash := sha256.New()
	n, err := hash.Write(message)
	if n != len(message) {
		return errors.Errorf("hash failure")
	}
	if err != nil {
		return err
	}
	digest := hash.Sum(nil)

	lowS, err := IsLowS(d.pk, signature.S)
	if err != nil {
		return err
	}
	if !lowS {
		return errors.New("signature is not in lowS")
	}

	valid := ecdsa.Verify(d.pk, digest, signature.R, signature.S)
	if !valid {
		return errors.Errorf("signature not valid")
	}

	return nil
}

// IsLowS checks that s is a low-S
func IsLowS(k *ecdsa.PublicKey, s *big.Int) (bool, error) {
	halfOrder, ok := curveHalfOrders[k.Curve]
	if !ok {
		return false, errors.Errorf("curve not recognized [%s]", k.Curve)
	}

	return s.Cmp(halfOrder) != 1, nil

}

// PemDecodeKey takes bytes and returns a Go key
func PemDecodeKey(keyBytes []byte) (interface{}, error) {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("bytes are not PEM encoded")
	}

	var key interface{}
	var err error
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "pem bytes are not PKCS8 encoded ")
		}
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "pem bytes are not cert encoded ")
		}
		return cert.PublicKey, nil
	case "PUBLIC KEY":
		key, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "pem bytes are not PKIX encoded ")
		}
	default:
		return nil, errors.Errorf("bad key type %s", block.Type)
	}

	return key, nil
}

func SignatureToLowS(k *ecdsa.PublicKey, signature []byte) ([]byte, error) {
	r, s, err := UnmarshalECDSASignature(signature)
	if err != nil {
		return nil, err
	}

	s, err = ToLowS(k, s)
	if err != nil {
		return nil, err
	}

	return MarshalECDSASignature(r, s)
}

func ToLowS(k *ecdsa.PublicKey, s *big.Int) (*big.Int, error) {
	lowS, err := IsLowS(k, s)
	if err != nil {
		return nil, err
	}

	if !lowS {
		// Set s to N - s that will be then in the lower part of signature space
		// less or equal to half order
		s.Sub(k.Params().N, s)

		return s, nil
	}

	return s, nil
}

func MarshalECDSASignature(r, s *big.Int) ([]byte, error) {
	return asn1.Marshal(ECDSASignature{r, s})
}

func UnmarshalECDSASignature(raw []byte) (*big.Int, *big.Int, error) {
	// Unmarshal
	sig := new(ECDSASignature)
	_, err := asn1.Unmarshal(raw, sig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed unmashalling signature [%s]", err)
	}

	// Validate sig
	if sig.R == nil {
		return nil, nil, errors.New("invalid signature, R must be different from nil")
	}
	if sig.S == nil {
		return nil, nil, errors.New("invalid signature, S must be different from nil")
	}

	if sig.R.Sign() != 1 {
		return nil, nil, errors.New("invalid signature, R must be larger than zero")
	}
	if sig.S.Sign() != 1 {
		return nil, nil, errors.New("invalid signature, S must be larger than zero")
	}

	return sig.R, sig.S, nil
}

// SanitizeECDSASignedCert checks that the signatures signing a cert
// is in low-S. This is checked against the public key of parentCert.
// If the signature is not in low-S, then a new certificate is generated
// that is equals to cert but the signature that is in low-S.
func SanitizeECDSASignedCert(cert *x509.Certificate, parentCert *x509.Certificate) (*x509.Certificate, error) {
	if cert == nil {
		return nil, errors.New("certificate must be different from nil")
	}
	if parentCert == nil {
		return nil, errors.New("parent certificate must be different from nil")
	}

	expectedSig, err := SignatureToLowS(parentCert.PublicKey.(*ecdsa.PublicKey), cert.Signature)
	if err != nil {
		return nil, err
	}

	// if sig == cert.Signature, nothing needs to be done
	if bytes.Equal(cert.Signature, expectedSig) {
		return cert, nil
	}
	// otherwise create a new certificate with the new signature

	// 1. Unmarshal cert.Raw to get an instance of certificate,
	//    the lower level interface that represent an x509 certificate
	//    encoding
	var newCert certificate
	newCert, err = certFromX509Cert(cert)
	if err != nil {
		return nil, err
	}

	// 2. Change the signature
	newCert.SignatureValue = asn1.BitString{Bytes: expectedSig, BitLength: len(expectedSig) * 8}
	newCert.Raw = nil

	// 3. marshal again newCert. Raw must be nil
	newRaw, err := asn1.Marshal(newCert)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling of the certificate failed")
	}

	// 4. parse newRaw to get an x509 certificate
	return x509.ParseCertificate(newRaw)
}

type validity struct {
	NotBefore, NotAfter time.Time
}

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

type certificate struct {
	Raw                asn1.RawContent
	TBSCertificate     tbsCertificate
	SignatureAlgorithm pkix.AlgorithmIdentifier
	SignatureValue     asn1.BitString
}

type tbsCertificate struct {
	Raw                asn1.RawContent
	Version            int `asn1:"optional,explicit,default:0,tag:0"`
	SerialNumber       *big.Int
	SignatureAlgorithm pkix.AlgorithmIdentifier
	Issuer             asn1.RawValue
	Validity           validity
	Subject            asn1.RawValue
	PublicKey          publicKeyInfo
	UniqueId           asn1.BitString   `asn1:"optional,tag:1"`
	SubjectUniqueId    asn1.BitString   `asn1:"optional,tag:2"`
	Extensions         []pkix.Extension `asn1:"optional,explicit,tag:3"`
}

func certFromX509Cert(cert *x509.Certificate) (certificate, error) {
	var newCert certificate
	_, err := asn1.Unmarshal(cert.Raw, &newCert)
	if err != nil {
		return certificate{}, errors.Wrap(err, "unmarshalling of the certificate failed")
	}
	return newCert, nil
}

// String returns a PEM representation of a certificate
func (c certificate) String() string {
	b, err := asn1.Marshal(c)
	if err != nil {
		return fmt.Sprintf("Failed marshaling cert: %v", err)
	}
	block := &pem.Block{
		Bytes: b,
		Type:  "CERTIFICATE",
	}
	b = pem.EncodeToMemory(block)
	return string(b)
}
