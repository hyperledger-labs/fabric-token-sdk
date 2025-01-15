/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
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
	return []driver.Identity{script.Recipient}, nil
}

func (t *TypedIdentityDeserializer) GetOwnerAuditInfo(id driver.Identity, typ string, raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	if typ != htlc.ScriptType {
		return nil, errors.Errorf("invalid type, got [%s], expected [%s]", typ, htlc.ScriptType)
	}
	script := &htlc.Script{}
	var err error
	err = json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal htlc script")
	}

	auditInfo := &ScriptInfo{}
	auditInfo.Sender, err = p.GetAuditInfo(script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for htlc script [%s]", id.String())
	}
	auditInfo.Recipient, err = p.GetAuditInfo(script.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for script [%s]", id.String())
	}

	auditInfoRaw, err := json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for script")
	}
	return [][]byte{auditInfoRaw}, nil
}

func (t *TypedIdentityDeserializer) GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	panic("not implemented")
}

type AuditDeserializer struct {
	AuditInfoDeserializer driver2.AuditInfoDeserializer
}

func NewAuditDeserializer(auditInfoDeserializer driver2.AuditInfoDeserializer) *AuditDeserializer {
	return &AuditDeserializer{AuditInfoDeserializer: auditInfoDeserializer}
}

func (a *AuditDeserializer) DeserializeAuditInfo(bytes []byte) (driver2.AuditInfo, error) {
	si := &ScriptInfo{}
	err := json.Unmarshal(bytes, si)
	if err != nil || (len(si.Sender) == 0 && len(si.Recipient) == 0) {
		return nil, errors.Errorf("ivalid audit info, failed unmarshal [%s][%d][%d]", string(bytes), len(si.Sender), len(si.Recipient))
	}
	if len(si.Recipient) == 0 {
		return nil, errors.Errorf("no recipient defined")
	}
	ai, err := a.AuditInfoDeserializer.DeserializeAuditInfo(si.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unamrshalling audit info [%s]", bytes)
	}
	return ai, nil
}
