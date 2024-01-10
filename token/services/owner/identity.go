/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package owner

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

const (
	SerializedIdentityType = "si"
)

// TypedIdentity encodes an owner of an identity
type TypedIdentity struct {
	// Type encodes the type of the owner (currently it can only be a SerializedIdentity)
	Type string `protobuf:"bytes,1,opt,name=type,json=type,proto3" json:"type,omitempty"`
	// Identity encodes the identity
	Identity []byte `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
}

func UnmarshallTypedIdentity(id view.Identity) (*TypedIdentity, error) {
	si := &TypedIdentity{}
	_, err := asn1.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	return si, nil
}

func MarshallTypedIdentity(o *TypedIdentity) (view.Identity, error) {
	raw, err := asn1.Marshal(*o)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	return raw, nil
}

type DeserializeVerifierProvider interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

// TypedIdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type TypedIdentityDeserializer struct {
	DeserializeVerifierProvider
}

func NewTypedIdentityDeserializer(dvp DeserializeVerifierProvider) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{
		DeserializeVerifierProvider: dvp,
	}
}

func (d *TypedIdentityDeserializer) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	si, err := UnmarshallTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	return d.DeserializeVerifierProvider.DeserializeVerifier(si.Identity)
}

func (d *TypedIdentityDeserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.Errorf("signer deserialization not supported")
}

func (d *TypedIdentityDeserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return "info not supported", nil
}
