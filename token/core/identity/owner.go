/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	SerializedIdentityType = "si"
)

type ByteStringer func([]byte) string

var (
	typeFormatters = map[string]ByteStringer{
		SerializedIdentityType: serializedIdentityToBytes,
	}
)

func RegisterTypeFormatter(t string, stringer ByteStringer) {
	typeFormatters[t] = stringer
}

// RawOwner encodes an owner of an identity
type RawOwner struct {
	// Type encodes the type of the owner (currently it can only be a SerializedIdentity)
	Type string `protobuf:"bytes,1,opt,name=type,json=type,proto3" json:"type,omitempty"`
	// Identity encodes the identity
	Identity []byte `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
}

func serializedIdentityToBytes(in []byte) string {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(in, si)
	if err != nil {
		return fmt.Sprintf("badly encoded identity (%v)", err)
	}
	block, _ := pem.Decode(si.IdBytes)
	if block == nil {
		return fmt.Sprintf("badly encoded PEM (%s)", base64.StdEncoding.EncodeToString(si.IdBytes))
	}
	if block.Type != "CERTIFICATE" {
		return fmt.Sprintf("PEM with invalid type (%s)", block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Sprintf("badly encoded certificate (%v)", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return fmt.Sprintf("badly encoded public key (%v)", err)
	}
	return fmt.Sprintf("{MSP: '%s', PubKey: '%s'}", si.Mspid, base64.StdEncoding.EncodeToString(pubKeyBytes))
}

func (r *RawOwner) Reset() {
	*r = RawOwner{}
}

func (r *RawOwner) String() string {
	formatter, exists := typeFormatters[r.Type]
	if !exists {
		return fmt.Sprintf("Owner with unknown type %s", r.Type)
	}
	return formatter(r.Identity)
}

func (r *RawOwner) ProtoMessage() {}

type VerifierProvider interface {
	GetVerifier(id view.Identity) (driver.Verifier, error)
}

// RawOwnerIdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type RawOwnerIdentityDeserializer struct {
	VerifierProvider
}

func NewRawOwnerIdentityDeserializer(verifierProvider VerifierProvider) *RawOwnerIdentityDeserializer {
	return &RawOwnerIdentityDeserializer{
		VerifierProvider: verifierProvider,
	}
}

func (deserializer *RawOwnerIdentityDeserializer) GetVerifier(id view.Identity) (driver.Verifier, error) {
	si := &RawOwner{}
	err := json.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	return deserializer.VerifierProvider.GetVerifier(si.Identity)
}

func (deserializer *RawOwnerIdentityDeserializer) DeserializeVerifier(raw []byte) (driver2.Verifier, error) {
	return deserializer.GetVerifier(raw)
}

func (deserializer *RawOwnerIdentityDeserializer) DeserializeSigner(raw []byte) (driver2.Signer, error) {
	return nil, errors.Errorf("signer deserialization not supported")
}

func (deserializer *RawOwnerIdentityDeserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return "info not supported", nil
}
