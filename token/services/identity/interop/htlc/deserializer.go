/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

type VerifierDES interface {
	DeserializeVerifier(id driver.Identity) (driver.Verifier, error)
}

type TypedIdentityDeserializer struct {
	VerifierDeserializer VerifierDES
}

func NewTypedIdentityDeserializer(verifierDeserializer VerifierDES) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{VerifierDeserializer: verifierDeserializer}
}

func (t *TypedIdentityDeserializer) DeserializeVerifier(typ string, raw []byte) (driver.Verifier, error) {
	if typ != htlc.ScriptType {
		return nil, errors.Errorf("cannot deserializer type [%s], expected [%s]", typ, htlc.ScriptType)
	}

	script := &htlc.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal TypedIdentity as an htlc script")
	}
	v := &htlc.Verifier{}
	v.Sender, err = t.VerifierDeserializer.DeserializeVerifier(script.Sender)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the sender in the htlc script")
	}
	v.Recipient, err = t.VerifierDeserializer.DeserializeVerifier(script.Recipient)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the recipient in the htlc script")
	}
	v.Deadline = script.Deadline
	v.HashInfo.Hash = script.HashInfo.Hash
	v.HashInfo.HashFunc = script.HashInfo.HashFunc
	v.HashInfo.HashEncoding = script.HashInfo.HashEncoding
	return v, nil
}

func (t *TypedIdentityDeserializer) Recipients(id driver.Identity, typ string, raw []byte) ([]driver.Identity, error) {
	if typ != htlc.ScriptType {
		return nil, errors.New("unknown identity type")
	}

	script := &htlc.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal htlc script")
	}
	return []driver.Identity{script.Sender, script.Recipient}, nil
}
