/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer(pp *crypto.PublicParams) (*Deserializer, error) {
	if pp == nil {
		return nil, errors.New("failed to get deserializer: nil public parameters")
	}
	idemixDes, err := idemix.NewDeserializer(pp.IdemixIssuerPK, pp.IdemixCurveID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params [%d]", pp.IdemixCurveID)
	}
	m := deserializer.NewTypedVerifierDeserializerMultiplex()
	m.AddTypedVerifierDeserializer(msp.X509Identity, deserializer.NewTypedIdentityVerifierDeserializer(&x509.MSPIdentityDeserializer{}, &x509.AuditMatcherDeserializer{}))
	m.AddTypedVerifierDeserializer(msp.IdemixIdentity, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
	m.AddTypedVerifierDeserializer(htlc2.ScriptType, htlc.NewTypedIdentityDeserializer(m))

	return &Deserializer{
		Deserializer: common.NewDeserializer(
			msp.IdemixIdentity,
			&x509.MSPIdentityDeserializer{},
			m,
			&x509.MSPIdentityDeserializer{},
			m,
			m,
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

func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, label string) (*crypto.PublicParams, error) {
	return crypto.NewPublicParamsFromBytes(raw, label)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token
type EIDRHDeserializer = deserializer.EIDRHDeserializer

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	d := deserializer.NewEIDRHDeserializer()
	d.AddDeserializer(msp.IdemixIdentity, &idemix.AuditInfoDeserializer{})
	d.AddDeserializer(msp.X509Identity, &x509.AuditInfoDeserializer{})
	d.AddDeserializer(htlc2.ScriptType, htlc.NewAuditDeserializer(&idemix.AuditInfoDeserializer{}))
	return d
}
