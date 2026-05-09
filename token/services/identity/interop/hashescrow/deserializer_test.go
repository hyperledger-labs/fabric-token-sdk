/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driverMock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	identityDriverMock "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver/mock"
	ihe "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/hashescrow"
	he "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/stretchr/testify/require"
)

type stubDeserializer struct {
	deserializeVerifier func(ctx context.Context, id driver.Identity) (driver.Verifier, error)
	matchIdentity       func(ctx context.Context, id driver.Identity, ai []byte) error
}

func (s *stubDeserializer) DeserializeVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	if s.deserializeVerifier != nil {
		return s.deserializeVerifier(ctx, id)
	}

	return &driverMock.Verifier{}, nil
}

func (s *stubDeserializer) MatchIdentity(ctx context.Context, id driver.Identity, ai []byte) error {
	if s.matchIdentity != nil {
		return s.matchIdentity(ctx, id, ai)
	}

	return nil
}

type fakeAuditInfo struct{}

func (f *fakeAuditInfo) EnrollmentID() string     { return "e" }
func (f *fakeAuditInfo) RevocationHandle() string { return "r" }

func mkScript(t *testing.T, sender, recipient []byte) []byte {
	t.Helper()
	s := &he.Script{
		Sender:    sender,
		Recipient: recipient,
		RecipientHashInfo: he.HashInfo{
			Hash: []byte("rh"),
		},
		SenderHashInfo: he.HashInfo{
			Hash: []byte("sh"),
		},
	}
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	return raw
}

func TestGetScriptSenderAndRecipient(t *testing.T) {
	raw := mkScript(t, []byte("s"), []byte("r"))
	sender, recipient, err := ihe.GetScriptSenderAndRecipient(raw)
	require.NoError(t, err)
	require.Equal(t, identity.Identity("s"), sender)
	require.Equal(t, identity.Identity("r"), recipient)

	_, _, err = ihe.GetScriptSenderAndRecipient([]byte("bad-json"))
	require.Error(t, err)
}

func TestTypedIdentityDeserializerDeserializeVerifier(t *testing.T) {
	d := ihe.NewTypedIdentityDeserializer(&stubDeserializer{})
	ctx := t.Context()

	_, err := d.DeserializeVerifier(ctx, identity.Type(999), []byte{})
	require.Error(t, err)

	_, err = d.DeserializeVerifier(ctx, he.ScriptType, []byte("bad-json"))
	require.Error(t, err)

	stub := &stubDeserializer{
		deserializeVerifier: func(context.Context, driver.Identity) (driver.Verifier, error) {
			return nil, errors.New("nope")
		},
	}
	d = ihe.NewTypedIdentityDeserializer(stub)
	_, err = d.DeserializeVerifier(ctx, he.ScriptType, mkScript(t, []byte("s"), []byte("r")))
	require.Error(t, err)

	d = ihe.NewTypedIdentityDeserializer(&stubDeserializer{})
	v, err := d.DeserializeVerifier(ctx, he.ScriptType, mkScript(t, []byte("s"), []byte("r")))
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestTypedIdentityDeserializerRecipientsAndAuditInfo(t *testing.T) {
	d := ihe.NewTypedIdentityDeserializer(&stubDeserializer{})
	ctx := t.Context()
	raw := mkScript(t, []byte("s"), []byte("r"))

	_, err := d.Recipients(nil, identity.Type(999), raw)
	require.Error(t, err)

	ids, err := d.Recipients(nil, he.ScriptType, raw)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, identity.Identity("r"), ids[0])

	p := &driverMock.AuditInfoProvider{}
	p.GetAuditInfoReturnsOnCall(0, nil, errors.New("nope"))
	_, err = d.GetAuditInfo(ctx, []byte("id"), he.ScriptType, raw, p)
	require.Error(t, err)

	p = &driverMock.AuditInfoProvider{}
	p.GetAuditInfoReturnsOnCall(0, []byte("sa"), nil)
	p.GetAuditInfoReturnsOnCall(1, []byte("ra"), nil)
	ai, err := d.GetAuditInfo(ctx, []byte("id"), he.ScriptType, raw, p)
	require.NoError(t, err)

	var out ihe.ScriptInfo
	require.NoError(t, json.Unmarshal(ai, &out))
	require.Equal(t, []byte("sa"), out.Sender)
	require.Equal(t, []byte("ra"), out.Recipient)
}

func TestAuditDeserializerAndMatcher(t *testing.T) {
	ctx := t.Context()
	aid := &identityDriverMock.AuditInfoDeserializer{}
	ad := ihe.NewAuditDeserializer(aid)

	scriptRaw, err := json.Marshal(&he.Script{Recipient: []byte("r")})
	require.NoError(t, err)

	_, err = ad.DeserializeAuditInfo(ctx, scriptRaw, []byte("bad-json"))
	require.Error(t, err)

	siRaw, err := json.Marshal(&ihe.ScriptInfo{Sender: []byte("s")})
	require.NoError(t, err)
	_, err = ad.DeserializeAuditInfo(ctx, scriptRaw, siRaw)
	require.Error(t, err)

	siRaw, err = json.Marshal(&ihe.ScriptInfo{Recipient: []byte("ra")})
	require.NoError(t, err)
	aid.DeserializeAuditInfoReturns(&fakeAuditInfo{}, nil)
	ai, err := ad.DeserializeAuditInfo(ctx, scriptRaw, siRaw)
	require.NoError(t, err)
	require.NotNil(t, ai)

	matcher := &ihe.AuditInfoMatcher{
		AuditInfo: []byte("bad-json"),
		Deserializer: &stubDeserializer{
			matchIdentity: func(context.Context, driver.Identity, []byte) error { return nil },
		},
	}
	require.Error(t, matcher.Match(ctx, []byte("id")))

	matcher.AuditInfo, err = json.Marshal(&ihe.ScriptInfo{Sender: []byte("sa"), Recipient: []byte("ra")})
	require.NoError(t, err)
	require.Error(t, matcher.Match(ctx, []byte("bad-script")))

	matcher.Deserializer = &stubDeserializer{
		matchIdentity: func(context.Context, driver.Identity, []byte) error { return errors.New("nope") },
	}
	require.Error(t, matcher.Match(ctx, mkScript(t, []byte("s"), []byte("r"))))

	matcher.Deserializer = &stubDeserializer{}
	require.NoError(t, matcher.Match(ctx, mkScript(t, []byte("s"), []byte("r"))))
}
