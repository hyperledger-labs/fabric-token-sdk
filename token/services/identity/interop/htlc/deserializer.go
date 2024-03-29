/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
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
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal TypedIdentity")
	}
	if si.Type == htlc.ScriptType {
		return d.getHTLCVerifier(si.Identity)
	}
	return d.OwnerDeserializer.DeserializeVerifier(id)
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
