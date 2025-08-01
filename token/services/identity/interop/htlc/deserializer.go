/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

type deserializer interface {
	DeserializeVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error)
	MatchIdentity(id driver.Identity, ai []byte) error
}

type TypedIdentityDeserializer struct {
	deserializer deserializer
}

func NewTypedIdentityDeserializer(deserializer deserializer) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{deserializer: deserializer}
}

func (t *TypedIdentityDeserializer) DeserializeVerifier(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error) {
	if typ != htlc.ScriptType {
		return nil, errors.Errorf("cannot deserializer type [%s], expected [%s]", typ, htlc.ScriptType)
	}

	script := &htlc.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal TypedIdentity as an htlc script")
	}
	v := &htlc.Verifier{}
	v.Sender, err = t.deserializer.DeserializeVerifier(ctx, script.Sender)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the sender in the htlc script")
	}
	v.Recipient, err = t.deserializer.DeserializeVerifier(ctx, script.Recipient)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the recipient in the htlc script")
	}
	v.Deadline = script.Deadline
	v.HashInfo.Hash = script.HashInfo.Hash
	v.HashInfo.HashFunc = script.HashInfo.HashFunc
	v.HashInfo.HashEncoding = script.HashInfo.HashEncoding
	return v, nil
}

func (t *TypedIdentityDeserializer) Recipients(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
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

func (t *TypedIdentityDeserializer) GetAuditInfo(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
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
	auditInfo.Sender, err = p.GetAuditInfo(ctx, script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for htlc script [%s]", id.String())
	}
	auditInfo.Recipient, err = p.GetAuditInfo(ctx, script.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for script [%s]", id.String())
	}

	auditInfoRaw, err := json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for script")
	}
	return auditInfoRaw, nil
}

func (t *TypedIdentityDeserializer) GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return &AuditInfoMatcher{
		auditInfo:    auditInfo,
		deserializer: t.deserializer,
	}, nil
}

type AuditDeserializer struct {
	AuditInfoDeserializer driver2.AuditInfoDeserializer
}

func NewAuditDeserializer(auditInfoDeserializer driver2.AuditInfoDeserializer) *AuditDeserializer {
	return &AuditDeserializer{AuditInfoDeserializer: auditInfoDeserializer}
}

func (a *AuditDeserializer) DeserializeAuditInfo(ctx context.Context, raw []byte) (driver2.AuditInfo, error) {
	si := &ScriptInfo{}
	err := json.Unmarshal(raw, si)
	if err != nil || (len(si.Sender) == 0 && len(si.Recipient) == 0) {
		return nil, errors.Errorf("ivalid audit info, failed unmarshal [%s][%d][%d]", string(raw), len(si.Sender), len(si.Recipient))
	}
	if len(si.Recipient) == 0 {
		return nil, errors.Errorf("no recipient defined")
	}
	ai, err := a.AuditInfoDeserializer.DeserializeAuditInfo(ctx, si.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unamrshalling audit info [%s]", raw)
	}
	return ai, nil
}

type AuditInfoMatcher struct {
	auditInfo    []byte
	deserializer deserializer
}

func (a *AuditInfoMatcher) Match(id []byte) error {
	scriptInf := &ScriptInfo{}
	if err := json.Unmarshal(a.auditInfo, scriptInf); err != nil {
		return errors.Wrapf(err, "failed to unmarshal script info")
	}
	scriptSender, scriptRecipient, err := GetScriptSenderAndRecipient(id)
	if err != nil {
		return errors.Wrap(err, "failed getting script sender and recipient")
	}
	err = a.deserializer.MatchIdentity(scriptSender, scriptInf.Sender)
	if err != nil {
		return errors.Wrapf(err, "failed matching sender identity [%s]", scriptSender.String())
	}
	err = a.deserializer.MatchIdentity(scriptRecipient, scriptInf.Recipient)
	if err != nil {
		return errors.Wrapf(err, "failed matching recipient identity [%s]", scriptRecipient.String())
	}
	return nil
}

// GetScriptSenderAndRecipient returns the script's sender and recipient according to the type of the given owner
func GetScriptSenderAndRecipient(id []byte) (sender, recipient driver.Identity, err error) {
	script := &htlc.Script{}
	err = json.Unmarshal(id, script)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to unmarshal htlc script")
	}
	return script.Sender, script.Recipient, nil
}
