/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package role

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type localMembership interface {
	DefaultNetworkIdentity() driver.Identity
	IsMe(id driver.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (idriver.IdentityInfo, error)
	GetIdentifier(id driver.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(config driver.IdentityConfiguration) error
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
func (r *Role) GetIdentityInfo(id string) (idriver.IdentityInfo, error) {
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] getting info for [%s]", r.networkID, id)
	}

	info, err := r.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "[%s] failed to get identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// RegisterIdentity registers the given identity
func (r *Role) RegisterIdentity(config driver.IdentityConfiguration) error {
	return r.localMembership.RegisterIdentity(config)
}

func (r *Role) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}

// MapToIdentity returns the identity for the given argument
func (r *Role) MapToIdentity(v driver.WalletLookupID) (driver.Identity, string, error) {
	switch vv := v.(type) {
	case driver.Identity:
		return r.mapIdentityToID(vv)
	case []byte:
		return r.mapIdentityToID(vv)
	case string:
		return r.mapStringToID(vv)
	default:
		return nil, "", errors.Errorf("identifier not recognised, expected []byte or driver.Identity, got [%T], [%s]", v, string(debug.Stack()))
	}
}

func (r *Role) mapStringToID(v string) (driver.Identity, string, error) {
	defaultNetworkIdentity := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] mapping string identifier for [%s,%s], default identities [%s:%s]",
			r.networkID,
			v,
			hash.Hashable(v).String(),
			defaultNetworkIdentity.String(),
			r.nodeIdentity.String(),
		)
	}

	label := v
	labelAsIdentity := driver.Identity(label)
	switch {
	case len(label) == 0:
		r.logger.Debugf("passed empty label")
		return nil, defaultIdentifier, nil
	case label == defaultIdentifier:
		r.logger.Debugf("passed default identifier")
		return nil, defaultIdentifier, nil
	case label == defaultNetworkIdentity.UniqueID():
		r.logger.Debugf("passed default identity")
		return nil, defaultIdentifier, nil
	case label == string(defaultNetworkIdentity):
		r.logger.Debugf("passed default identity as string")
		return nil, defaultIdentifier, nil
	case defaultNetworkIdentity.Equal(labelAsIdentity):
		r.logger.Debugf("passed default identity as view identity")
		return nil, defaultIdentifier, nil
	case r.nodeIdentity.Equal(labelAsIdentity):
		r.logger.Debugf("passed node identity as view identity")
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(labelAsIdentity):
		r.logger.Debugf("passed a local member")
		id := labelAsIdentity
		if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
			return nil, idIdentifier, nil
		}
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("failed getting identity info for [%s], returning the identity", id)
		}
		return id, "", nil
	}

	if idIdentifier, err := r.localMembership.GetIdentifier(labelAsIdentity); err == nil {
		return nil, idIdentifier, nil
	}
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("cannot find match for string [%s]", v)
	}
	return nil, label, nil
}

func (r *Role) mapIdentityToID(v driver.Identity) (driver.Identity, string, error) {
	defaultNetworkIdentity := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] mapping driver.Identity identifier for [%s], default identities [%s:%s]",
			r.networkID,
			v,
			defaultNetworkIdentity.String(),
			r.nodeIdentity.String(),
		)
	}

	id := v
	switch {
	case id.IsNone():
		r.logger.Debugf("passed empty identity")
		return nil, defaultIdentifier, nil
	case id.Equal(defaultNetworkIdentity):
		r.logger.Debugf("passed default identity")
		return nil, defaultIdentifier, nil
	case string(id) == defaultIdentifier:
		r.logger.Debugf("passed default identifier")
		return nil, defaultIdentifier, nil
	case id.Equal(r.nodeIdentity):
		r.logger.Debugf("passed identity is the node identity (same bytes)")
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(id):
		r.logger.Debugf("passed identity is me")
		if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
			return id, idIdentifier, nil
		}
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("failed getting identity info for [%s], returning the identity", id)
		}
		return id, "", nil
	}
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("looking up identifier for identity as label [%s]", hash.Hashable(id))
	}

	label := string(id)
	if info, err := r.localMembership.GetIdentityInfo(label, nil); err == nil {
		return nil, info.ID(), nil
	}
	if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
		return nil, idIdentifier, nil
	}

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("cannot find match for driver.Identity string [%s]", id)
	}
	return nil, string(id), nil
}
