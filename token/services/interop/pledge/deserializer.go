/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
)

// ScriptInfo includes info about the sender and the recipient
type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

func (si *ScriptInfo) Marshal() ([]byte, error) {
	return json.Marshal(si)
}

func (si *ScriptInfo) Unmarshal(raw []byte) error {
	return json.Unmarshal(raw, si)
}

type VerifierDES interface {
	DeserializeVerifier(id token.Identity) (token.Verifier, error)
}

type TypedIdentityDeserializer struct {
	VerifierDeserializer VerifierDES
}

func NewTypedIdentityDeserializer(verifierDeserializer VerifierDES) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{VerifierDeserializer: verifierDeserializer}
}

func (t *TypedIdentityDeserializer) DeserializeVerifier(typ string, raw []byte) (token.Verifier, error) {
	if typ != ScriptType {
		return nil, errors.Errorf("cannot deserializer type [%s], expected [%s]", typ, ScriptType)
	}

	script := &Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal TypedIdentity as an htlc script")
	}
	v := &Verifier{}
	v.Sender, err = t.VerifierDeserializer.DeserializeVerifier(script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal the identity of the sender [%v]", script.Sender.String())
	}
	v.Issuer, err = t.VerifierDeserializer.DeserializeVerifier(script.Issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal the identity of the issuer [%s]", script.Issuer.String())
	}
	v.PledgeID = script.ID
	return v, nil
}

func (t *TypedIdentityDeserializer) Recipients(id token.Identity, typ string, raw []byte) ([]token.Identity, error) {
	logger.Infof("pledge, get recipients for [%s][%s]", id, typ)
	if typ != ScriptType {
		return nil, errors.New("unknown identity type")
	}

	script := &Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal htlc script")
	}
	return []token.Identity{script.Issuer}, nil
}

func (t *TypedIdentityDeserializer) GetOwnerAuditInfo(id token.Identity, typ string, raw []byte, p deserializer.AuditInfoProvider) ([][]byte, error) {
	logger.Infof("1. pledge, get owner audit info for [%s][%s]", id, typ)
	if typ != ScriptType {
		return nil, errors.Errorf("invalid type, got [%s], expected [%s]", typ, ScriptType)
	}
	script := &Script{}
	var err error
	err = json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal htlc script")
	}

	logger.Infof("2. pledge, get owner audit info for [%s][%s]", id, typ)
	auditInfo := &ScriptInfo{}
	auditInfo.Sender, err = p.GetAuditInfo(script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for sender of pledge script [%s]", view.Identity(raw).String())
	}

	logger.Infof("3. pledge, get owner audit info for [%s][%s]", id, typ)
	if len(auditInfo.Sender) == 0 { // in case this is a redeem we need to check the script issuer (and not the script sender)
		auditInfo.Sender, err = p.GetAuditInfo(script.Issuer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for issuer of pledge script [%s]", view.Identity(raw).String())
		}
		if len(auditInfo.Sender) == 0 {
			return nil, errors.Errorf("failed getting audit info for pledge script [%s]", view.Identity(raw).String())
		}
	}

	logger.Infof("4. pledge, get owner audit info for [%s][%s]", id, typ)
	// Notice that recipient is in another network, but the issuer is
	// the actual recipient of the script because it is in the same network.
	auditInfo.Recipient, err = p.GetAuditInfo(script.Issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for issuer of pledge script [%s]", view.Identity(raw).String())
	}

	logger.Infof("5. pledge, get owner audit info for [%s][%s] [%s]", id, typ, debug.Stack())
	auditInfoRaw, err := json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for script")
	}
	return [][]byte{auditInfoRaw}, nil
}

type AuditDeserializer struct {
	AuditInfoDeserializer driver.AuditInfoDeserializer
}

func NewAuditDeserializer(auditInfoDeserializer driver.AuditInfoDeserializer) *AuditDeserializer {
	return &AuditDeserializer{AuditInfoDeserializer: auditInfoDeserializer}
}

func (a *AuditDeserializer) DeserializeAuditInfo(bytes []byte) (driver.AuditInfo, error) {
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

// GetScriptSenderAndRecipient returns the script's sender, recipient, and issuer
func GetScriptSenderAndRecipient(ro *identity.TypedIdentity) (sender, recipient, issuer token.Identity, err error) {
	if ro.Type == ScriptType {
		script := &Script{}
		err = json.Unmarshal(ro.Identity, script)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal htlc script")
		}
		return script.Sender, script.Recipient, script.Issuer, nil
	}
	return nil, nil, nil, errors.New("unknown identity type")
}
