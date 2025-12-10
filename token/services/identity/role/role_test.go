/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package role_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIdentityInfo is a minimal implementation of idriver.IdentityInfo used in tests
type fakeIdentityInfo struct {
	id string
}

func (f *fakeIdentityInfo) ID() string           { return f.id }
func (f *fakeIdentityInfo) EnrollmentID() string { return "" }
func (f *fakeIdentityInfo) Remote() bool         { return false }
func (f *fakeIdentityInfo) Get(ctx context.Context) (driver.Identity, []byte, error) {
	return driver.Identity(f.id), nil, nil
}
func (f *fakeIdentityInfo) Anonymous() bool { return false }

// ensure tests compile even if some methods are not used
var _ idriver.IdentityInfo = &fakeIdentityInfo{}

func setup(t *testing.T) (context.Context, *role.Role, *mock.LocalMembership) {
	t.Helper()
	ctx := t.Context()
	logger := logging.MustGetLogger("role_test")
	m := &mock.LocalMembership{}
	// default values for network identity and identifiers
	m.DefaultNetworkIdentityReturns(driver.Identity("defaultNet"))
	m.GetDefaultIdentifierReturns("defaultID")

	r := role.NewRole(logger, identity.IssuerRole, "net1", driver.Identity("nodeID"), m)
	return ctx, r, m
}

func TestRole_ID_returns_roleID(t *testing.T) {
	_, r, _ := setup(t)
	require.EqualValues(t, identity.IssuerRole, r.ID())
}

func TestRole_GetIdentityInfo_success_and_error(t *testing.T) {
	ctx, r, m := setup(t)
	info := &fakeIdentityInfo{id: "info1"}
	m.GetIdentityInfoReturns(info, nil)

	got, err := r.GetIdentityInfo(ctx, "label")
	require.NoError(t, err)
	require.Equal(t, info, got)

	// error path
	m.GetIdentityInfoReturns(nil, errors.New("not found"))
	_, err = r.GetIdentityInfo(ctx, "label")
	require.Error(t, err)
}

func TestRole_RegisterIdentity_and_IdentityIDs(t *testing.T) {
	ctx, r, m := setup(t)
	m.RegisterIdentityReturns(nil)
	err := r.RegisterIdentity(ctx, driver.IdentityConfiguration{})
	require.NoError(t, err)

	m.RegisterIdentityReturns(errors.New("bad"))
	err = r.RegisterIdentity(ctx, driver.IdentityConfiguration{})
	require.Error(t, err)

	// IdentityIDs
	m.IDsReturns([]string{"a", "b"}, nil)
	ids, err := r.IdentityIDs()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"a", "b"}, ids)

	m.IDsReturns(nil, errors.New("fail"))
	_, err = r.IdentityIDs()
	require.Error(t, err)
}

func TestRole_MapToIdentity_unsupported_type(t *testing.T) {
	ctx, r, _ := setup(t)
	_, _, err := r.MapToIdentity(ctx, 123)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "identifier not recognised")
}

func TestRole_MapToIdentity_mapStringToID_branches(t *testing.T) {
	ctx, r, m := setup(t)
	// Use stubs to control GetIdentifier and IsMe behaviour based on input
	m.GetIdentifierCalls(func(ctx context.Context, id driver.Identity) (string, error) {
		s := string(id)
		switch s {
		case "label":
			return "id123", nil
		case "member":
			return "idMe", nil
		case "member2":
			return "", errors.New("no")
		default:
			return "", errors.New("no")
		}
	})
	m.IsMeCalls(func(ctx context.Context, id driver.Identity) bool {
		s := string(id)
		return s == "member" || s == "member2"
	})

	// If GetIdentifier succeeds immediately
	id, ident, err := r.MapToIdentity(ctx, "label")
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "id123", ident)

	// empty label -> default identifier
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, "")
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// passed default identifier
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, "def")
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// passed default network identity unique id
	m.DefaultNetworkIdentityReturns(driver.Identity("uniq"))
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, "uniq")
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// passed default network identity as string
	m.DefaultNetworkIdentityReturns(driver.Identity("netstr"))
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, string(driver.Identity("netstr")))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// passed node identity equal
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, string(driver.Identity("nodeID")))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// is local member and GetIdentifier succeeds
	id, ident, err = r.MapToIdentity(ctx, "member")
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "idMe", ident)

	// is local member and GetIdentifier fails -> return identity
	id, ident, err = r.MapToIdentity(ctx, "member2")
	require.NoError(t, err)
	require.Equal(t, driver.Identity("member2"), id)
	require.Equal(t, "", ident)

	// fallback: return label as identifier
	// make IsMe return false for unknown
	m.IsMeCalls(func(ctx context.Context, id driver.Identity) bool { return false })
	id, ident, err = r.MapToIdentity(ctx, "unknown")
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "unknown", ident)
}

func TestRole_MapToIdentity_mapIdentityToID_branches(t *testing.T) {
	ctx, r, m := setup(t)
	// Use stubs for IsMe, GetIdentifier and GetIdentityInfo
	m.IsMeCalls(func(ctx context.Context, id driver.Identity) bool {
		s := string(id)
		return s == "me" || s == "me2"
	})
	m.GetIdentifierCalls(func(ctx context.Context, id driver.Identity) (string, error) {
		s := string(id)
		switch s {
		case "me":
			return "idMe", nil
		case "me2":
			return "", errors.New("no")
		case "labelFallback":
			return "idFallback", nil
		default:
			return "", errors.New("no")
		}
	})
	m.GetIdentityInfoCalls(func(ctx context.Context, label string, auditInfo []byte) (idriver.IdentityInfo, error) {
		switch label {
		case "labelInfo":
			return &fakeIdentityInfo{id: "infoID"}, nil
		default:
			return nil, errors.New("no")
		}
	})

	// identity is none -> default
	m.GetDefaultIdentifierReturns("def")
	id, ident, err := r.MapToIdentity(ctx, driver.Identity(""))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// identity equals default network identity
	m.DefaultNetworkIdentityReturns(driver.Identity("n1"))
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, driver.Identity("n1"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// identity string equals default identifier
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, driver.Identity("def"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// identity equals nodeIdentity
	m.GetDefaultIdentifierReturns("def")
	id, ident, err = r.MapToIdentity(ctx, driver.Identity("nodeID"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "def", ident)

	// is me and GetIdentifier succeeds
	idVal := driver.Identity("me")
	id, ident, err = r.MapToIdentity(ctx, idVal)
	require.NoError(t, err)
	require.Equal(t, idVal, id)
	require.Equal(t, "idMe", ident)

	// is me and GetIdentifier fails -> return identity
	idVal2 := driver.Identity("me2")
	id, ident, err = r.MapToIdentity(ctx, idVal2)
	require.NoError(t, err)
	require.Equal(t, idVal2, id)
	require.Equal(t, "", ident)

	// lookup identity as label via GetIdentityInfo succeeds
	id, ident, err = r.MapToIdentity(ctx, driver.Identity("labelInfo"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "infoID", ident)

	// GetIdentityInfo fails, GetIdentifier succeeds
	id, ident, err = r.MapToIdentity(ctx, driver.Identity("labelFallback"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "idFallback", ident)

	// both fail -> return string(id)
	id, ident, err = r.MapToIdentity(ctx, driver.Identity("labelNo"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "labelNo", ident)

	// []byte path: should behave like driver.Identity
	id, ident, err = r.MapToIdentity(ctx, []byte("labelNo"))
	require.NoError(t, err)
	require.Nil(t, id)
	require.Equal(t, "labelNo", ident)
}
