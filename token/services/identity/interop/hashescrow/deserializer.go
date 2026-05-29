/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
)

const (
	ScriptType       = hashescrow.ScriptType
	ScriptTypeString = hashescrow.ScriptTypeString
)

//go:generate counterfeiter -o mock/deserializer.go -fake-name Deserializer . Deserializer
type Deserializer interface {
	DeserializeVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error)
	MatchIdentity(ctx context.Context, id driver.Identity, ai []byte) error
}

type TypedIdentityDeserializer struct {
	deserializer Deserializer
}

func NewTypedIdentityDeserializer(deserializer Deserializer) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{deserializer: deserializer}
}

func (t *TypedIdentityDeserializer) DeserializeVerifier(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error) {
	if typ != ScriptType {
		return nil, errors.Errorf("cannot deserializer type [%s], expected [%s]", typ, ScriptType)
	}

	script := &hashescrow.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal TypedIdentity as a hash escrow script")
	}
	// Ensure sender and recipient identities are still syntactically valid and deserializable.
	if _, err = t.deserializer.DeserializeVerifier(ctx, script.Sender); err != nil {
		return nil, errors.Errorf("failed to deserialize the identity of the sender in the hash escrow script")
	}
	if _, err = t.deserializer.DeserializeVerifier(ctx, script.Recipient); err != nil {
		return nil, errors.Errorf("failed to deserialize the identity of the recipient in the hash escrow script")
	}
	v := &hashescrow.Verifier{Script: script}

	return v, nil
}

func (t *TypedIdentityDeserializer) Recipients(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
	if typ != ScriptType {
		return nil, errors.New("unknown identity type")
	}
	script := &hashescrow.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal hash escrow script")
	}

	return []driver.Identity{script.Sender, script.Recipient}, nil
}

func (t *TypedIdentityDeserializer) GetAuditInfo(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
	if typ != ScriptType {
		return nil, errors.Errorf("invalid type, got [%s], expected [%s]", typ, ScriptType)
	}
	script := &hashescrow.Script{}
	if err := json.Unmarshal(raw, script); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal hash escrow script")
	}

	auditInfo := &ScriptInfo{}
	var err error
	auditInfo.Sender, err = p.GetAuditInfo(ctx, script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for hash escrow script [%s]", id.String())
	}
	auditInfo.Recipient, err = p.GetAuditInfo(ctx, script.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for hash escrow script [%s]", id.String())
	}

	auditInfoRaw, err := json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for hash escrow script")
	}

	return auditInfoRaw, nil
}

func (t *TypedIdentityDeserializer) GetAuditInfoMatcher(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return &AuditInfoMatcher{
		AuditInfo:    auditInfo,
		Deserializer: t.deserializer,
	}, nil
}

type AuditDeserializer struct {
	AuditInfoDeserializer idriver.AuditInfoDeserializer
}

func NewAuditDeserializer(auditInfoDeserializer idriver.AuditInfoDeserializer) *AuditDeserializer {
	return &AuditDeserializer{AuditInfoDeserializer: auditInfoDeserializer}
}

func (a *AuditDeserializer) DeserializeAuditInfo(ctx context.Context, identity driver.Identity, raw []byte) (idriver.AuditInfo, error) {
	script := &hashescrow.Script{}
	err := json.Unmarshal(identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal hash escrow identity")
	}

	si := &ScriptInfo{}
	err = json.Unmarshal(raw, si)
	if err != nil || (len(si.Sender) == 0 && len(si.Recipient) == 0) {
		return nil, errors.Errorf("invalid audit info, failed unmarshal [%s][%d][%d]", string(raw), len(si.Sender), len(si.Recipient))
	}
	if len(si.Recipient) == 0 {
		return nil, errors.Errorf("no recipient defined")
	}
	ai, err := a.AuditInfoDeserializer.DeserializeAuditInfo(ctx, script.Recipient, si.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling audit info [%s]", raw)
	}

	return ai, nil
}

type AuditInfoMatcher struct {
	AuditInfo    []byte
	Deserializer Deserializer
}

func (a *AuditInfoMatcher) Match(ctx context.Context, id []byte) error {
	scriptInf := &ScriptInfo{}
	if err := json.Unmarshal(a.AuditInfo, scriptInf); err != nil {
		return errors.Wrapf(err, "failed to unmarshal script info")
	}
	scriptSender, scriptRecipient, err := GetScriptSenderAndRecipient(id)
	if err != nil {
		return errors.Wrap(err, "failed getting script sender and recipient")
	}
	err = a.Deserializer.MatchIdentity(ctx, scriptSender, scriptInf.Sender)
	if err != nil {
		return errors.Wrapf(err, "failed matching sender identity [%s]", scriptSender.String())
	}
	err = a.Deserializer.MatchIdentity(ctx, scriptRecipient, scriptInf.Recipient)
	if err != nil {
		return errors.Wrapf(err, "failed matching recipient identity [%s]", scriptRecipient.String())
	}

	return nil
}

func GetScriptSenderAndRecipient(id []byte) (sender, recipient driver.Identity, err error) {
	script := &hashescrow.Script{}
	err = json.Unmarshal(id, script)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to unmarshal hash escrow script")
	}

	return script.Sender, script.Recipient, nil
}
