/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package role

import (
	"context"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
)

type localMembership interface {
	DefaultNetworkIdentity() driver.Identity
	IsMe(ctx context.Context, id driver.Identity) bool
	GetIdentityInfo(ctx context.Context, label string, auditInfo []byte) (idriver.IdentityInfo, error)
	GetIdentifier(ctx context.Context, id driver.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(ctx context.Context, config driver.IdentityConfiguration) error
	IDs() ([]string, error)
}

// Role models a role whose identities are anonymous
type Role struct {
	logger          logging.Logger
	roleID          identity.RoleType
	networkID       string
	localMembership localMembership
	nodeIdentity    driver.Identity
}

func NewRole(logger logging.Logger, roleID identity.RoleType, networkID string, nodeIdentity driver.Identity, localMembership localMembership) *Role {
	return &Role{
		logger:          logger,
		roleID:          roleID,
		networkID:       networkID,
		localMembership: localMembership,
		nodeIdentity:    nodeIdentity,
	}
}

func (r *Role) ID() identity.RoleType {
	return r.roleID
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (r *Role) GetIdentityInfo(ctx context.Context, id string) (idriver.IdentityInfo, error) {
	r.logger.DebugfContext(ctx, "[%s] getting info for [%s]", r.networkID, logging.Printable(id))

	info, err := r.localMembership.GetIdentityInfo(ctx, id, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "[%s] failed to get identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// RegisterIdentity registers the given identity
func (r *Role) RegisterIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return r.localMembership.RegisterIdentity(ctx, config)
}

func (r *Role) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}

// MapToIdentity returns the identity for the given argument
func (r *Role) MapToIdentity(ctx context.Context, v driver.WalletLookupID) (driver.Identity, string, error) {
	switch vv := v.(type) {
	case driver.Identity:
		return r.mapIdentityToID(ctx, vv)
	case []byte:
		return r.mapIdentityToID(ctx, vv)
	case string:
		return r.mapStringToID(ctx, vv)
	default:
		return nil, "", errors.Errorf("identifier not recognised, expected []byte or driver.Identity, got [%T], [%s]", v, string(debug.Stack()))
	}
}

func (r *Role) mapStringToID(ctx context.Context, v string) (driver.Identity, string, error) {
	defaultNetworkIdentity := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	r.logger.DebugfContext(ctx, "[%s] mapping string identifier for [%s,%s], default identities [%s:%s]",
		r.networkID,
		logging.Printable(v),
		utils.Hashable(v),
		defaultNetworkIdentity,
		r.nodeIdentity,
	)

	label := v
	labelAsIdentity := driver.Identity(label)

	// check immediately if there is an identifier with that label
	if idIdentifier, err := r.localMembership.GetIdentifier(ctx, labelAsIdentity); err == nil {
		return nil, idIdentifier, nil
	}

	switch {
	case len(label) == 0:
		r.logger.DebugfContext(ctx, "passed empty label")
		return nil, defaultIdentifier, nil
	case label == defaultIdentifier:
		r.logger.DebugfContext(ctx, "passed default identifier")
		return nil, defaultIdentifier, nil
	case label == defaultNetworkIdentity.UniqueID():
		r.logger.DebugfContext(ctx, "passed default identity")
		return nil, defaultIdentifier, nil
	case label == string(defaultNetworkIdentity):
		r.logger.DebugfContext(ctx, "passed default identity as string")
		return nil, defaultIdentifier, nil
	case defaultNetworkIdentity.Equal(labelAsIdentity):
		r.logger.DebugfContext(ctx, "passed default identity as view identity")
		return nil, defaultIdentifier, nil
	case r.nodeIdentity.Equal(labelAsIdentity):
		r.logger.DebugfContext(ctx, "passed node identity as view identity")
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(ctx, labelAsIdentity):
		r.logger.DebugfContext(ctx, "passed a local member")
		id := labelAsIdentity
		if idIdentifier, err := r.localMembership.GetIdentifier(ctx, id); err == nil {
			return nil, idIdentifier, nil
		}
		r.logger.DebugfContext(ctx, "failed getting identity info for [%s], returning the identity", id)
		return id, "", nil
	}

	r.logger.DebugfContext(ctx, "cannot find match for string [%s]", v)
	return nil, label, nil
}

func (r *Role) mapIdentityToID(ctx context.Context, v driver.Identity) (driver.Identity, string, error) {
	defaultNetworkIdentity := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	r.logger.DebugfContext(ctx, "[%s] mapping driver.Identity identifier for [%s], default identities [%s:%s]",
		r.networkID,
		v,
		defaultNetworkIdentity,
		r.nodeIdentity,
	)

	id := v
	switch {
	case id.IsNone():
		r.logger.DebugfContext(ctx, "passed empty identity")
		return nil, defaultIdentifier, nil
	case id.Equal(defaultNetworkIdentity):
		r.logger.DebugfContext(ctx, "passed default identity")
		return nil, defaultIdentifier, nil
	case string(id) == defaultIdentifier:
		r.logger.DebugfContext(ctx, "passed default identifier")
		return nil, defaultIdentifier, nil
	case id.Equal(r.nodeIdentity):
		r.logger.DebugfContext(ctx, "passed identity is the node identity (same bytes)")
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(ctx, id):
		r.logger.DebugfContext(ctx, "passed identity is me")
		if idIdentifier, err := r.localMembership.GetIdentifier(ctx, id); err == nil {
			return id, idIdentifier, nil
		}
		r.logger.DebugfContext(ctx, "failed getting identity info for [%s], returning the identity", id)
		return id, "", nil
	}
	r.logger.DebugfContext(ctx, "looking up identifier for identity as label [%s]", utils.Hashable(id))

	label := string(id)
	if info, err := r.localMembership.GetIdentityInfo(ctx, label, nil); err == nil {
		return nil, info.ID(), nil
	}
	if idIdentifier, err := r.localMembership.GetIdentifier(ctx, id); err == nil {
		return nil, idIdentifier, nil
	}

	r.logger.DebugfContext(ctx, "cannot find match for driver.Identity string [%s]", id)
	return nil, string(id), nil
}
