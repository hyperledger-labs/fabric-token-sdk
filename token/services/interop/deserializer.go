/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

type Deserializer struct {
	OwnerDeserializer VerifierDES
}

func NewDeserializer(ownerDeserializer VerifierDES) *Deserializer {
	return &Deserializer{OwnerDeserializer: ownerDeserializer}
}

func (d *Deserializer) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	si, err := owner.UnmarshallTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal TypedIdentity")
	}

	switch t := si.Type; t {
	case owner.SerializedIdentityType:
		return d.OwnerDeserializer.DeserializeVerifier(id)
	case pledge.ScriptType:
		return d.getPledgeVerifier(si.Identity)
	case htlc.ScriptType:
		return d.getHTLCVerifier(si.Identity)
	default:
		return nil, errors.Errorf("failed to deserialize TypedIdentity: Unknown owner type %s", t)
	}
}

func (d *Deserializer) getPledgeVerifier(raw []byte) (driver.Verifier, error) {
	script := &pledge.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal TypedIdentity as a pledge script")
	}
	v := &pledge.Verifier{}
	v.Sender, err = d.OwnerDeserializer.DeserializeVerifier(script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal the identity of the sender [%v]", script.Sender.String())
	}
	v.Issuer, err = d.OwnerDeserializer.DeserializeVerifier(script.Issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal the identity of the issuer [%s]", script.Issuer.String())
	}
	v.PledgeID = script.ID
	return v, nil
}

func (d *Deserializer) getHTLCVerifier(raw []byte) (driver.Verifier, error) {
	script := &htlc.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal TypedIdentity as an htlc script")
	}
	v := &htlc.Verifier{}
	v.Sender, err = d.OwnerDeserializer.DeserializeVerifier(script.Sender)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the sender in the htlc script")
	}
	v.Recipient, err = d.OwnerDeserializer.DeserializeVerifier(script.Recipient)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the recipient in the htlc script")
	}
	v.Deadline = script.Deadline
	v.HashInfo.Hash = script.HashInfo.Hash
	v.HashInfo.HashFunc = script.HashInfo.HashFunc
	v.HashInfo.HashEncoding = script.HashInfo.HashEncoding
	return v, nil
}
