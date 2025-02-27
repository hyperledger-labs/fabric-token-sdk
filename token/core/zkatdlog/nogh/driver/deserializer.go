/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer(pp *v1.PublicParams) (*Deserializer, error) {
	if pp == nil {
		return nil, errors.New("failed to get deserializer: nil public parameters")
	}

	ownerDeserializer := deserializer.NewTypedVerifierDeserializerMultiplex()
	for _, idemixIssuerPublicKey := range pp.IdemixIssuerPublicKeys {
		idemixDes, err := idemix2.NewDeserializer(idemixIssuerPublicKey.PublicKey, idemixIssuerPublicKey.Curve)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params [%d]", idemixIssuerPublicKey.Curve)
		}
		ownerDeserializer.AddTypedVerifierDeserializer(idemix2.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		ownerDeserializer.AddTypedVerifierDeserializer(multisig.Multisig, multisig.NewTypedIdentityDeserializer(ownerDeserializer, ownerDeserializer))
	}
	ownerDeserializer.AddTypedVerifierDeserializer(x509.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&x509.IdentityDeserializer{}, &x509.AuditMatcherDeserializer{}))
	ownerDeserializer.AddTypedVerifierDeserializer(htlc2.ScriptType, htlc.NewTypedIdentityDeserializer(ownerDeserializer))

	auditorIssuerDeserializer := deserializer.NewTypedVerifierDeserializerMultiplex()
	auditorIssuerDeserializer.AddTypedVerifierDeserializer(x509.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&x509.IdentityDeserializer{}, &x509.AuditMatcherDeserializer{}))

	return &Deserializer{
		Deserializer: common.NewDeserializer(
			idemix2.IdentityType,
			auditorIssuerDeserializer,
			ownerDeserializer,
			auditorIssuerDeserializer,
			ownerDeserializer,
			ownerDeserializer,
		),
	}, nil
}

type TokenDeserializer struct{}

func (d *TokenDeserializer) DeserializeMetadata(raw []byte) (*token.Metadata, error) {
	metadata := &token.Metadata{}
	if err := metadata.Deserialize(raw); err != nil {
		return nil, err
	}
	return metadata, nil
}

func (d *TokenDeserializer) DeserializeToken(raw []byte) (*token.Token, error) {
	token := &token.Token{}
	if err := token.Deserialize(raw); err != nil {
		return nil, err
	}
	return token, nil
}

type PublicParamsDeserializer struct{}

func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, label string) (*v1.PublicParams, error) {
	return v1.NewPublicParamsFromBytes(raw, label)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token
type EIDRHDeserializer = deserializer.EIDRHDeserializer

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	d := deserializer.NewEIDRHDeserializer()
	d.AddDeserializer(idemix2.IdentityType, &idemix2.AuditInfoDeserializer{})
	d.AddDeserializer(x509.IdentityType, &x509.AuditInfoDeserializer{})
	d.AddDeserializer(htlc2.ScriptType, htlc.NewAuditDeserializer(&idemix2.AuditInfoDeserializer{}))
	d.AddDeserializer(multisig.Multisig, &multisig.AuditInfoDeserializer{})
	return d
}
