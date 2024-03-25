/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type localMembership interface {
	DefaultNetworkIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error)
	GetIdentifier(id view.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(config driver.IdentityConfiguration) error
	IDs() ([]string, error)
}

// Role is a container of idemix-based long-term identities.
type Role struct {
	roleID          driver.IdentityRole
	networkID       string
	nodeIdentity    view.Identity
	localMembership localMembership
}

func NewRole(roleID driver.IdentityRole, networkID string, nodeIdentity view.Identity, localMembership localMembership) *Role {
	return &Role{
		roleID:          roleID,
		networkID:       networkID,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

func (r *Role) ID() driver.IdentityRole {
	return r.roleID
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (r *Role) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] getting info for [%s]", r.networkID, id)
	}

	info, err := r.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "[%s] failed to get identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// MapToID returns the identity for the given argument
func (r *Role) MapToID(v driver.WalletLookupID) (view.Identity, string, error) {
	switch vv := v.(type) {
	case []byte:
		return r.mapIdentityToID(vv)
	case view.Identity:
		return r.mapIdentityToID(vv)
	case string:
		return r.mapStringToID(vv)
	default:
		return nil, "", errors.Errorf("identifier not recognised, expected []byte or view.Identity, got [%T], [%s]", v, string(debug.Stack()))
	}
}

func (r *Role) mapStringToID(v string) (view.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping string identifier for [%s,%s], default identities [%s:%s]",
			r.networkID,
			v,
			hash.Hashable(v).String(),
			defaultID.String(),
			r.nodeIdentity.String(),
		)
	}

	label := v
	viewIdentity := view.Identity(label)
	switch {
	case len(label) == 0:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed empty identity")
		}
		return nil, defaultIdentifier, nil
	case label == defaultIdentifier:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed default identifier")
		}
		return nil, defaultIdentifier, nil
	case label == defaultID.UniqueID():
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed default identity")
		}
		return nil, defaultIdentifier, nil
	case label == string(defaultID):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed default identity as string")
		}
		return nil, defaultIdentifier, nil
	case defaultID.Equal(viewIdentity):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed default identity as view identity")
		}
		return nil, defaultIdentifier, nil
	case r.nodeIdentity.Equal(viewIdentity):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed node identity as view identity")
		}
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(viewIdentity):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed a local member")
		}
		return nil, defaultIdentifier, nil
	}

	if idIdentifier, err := r.localMembership.GetIdentifier(viewIdentity); err == nil {
		return nil, idIdentifier, nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("cannot find match for string [%s]", v)
	}
	return nil, label, nil
}

func (r *Role) mapIdentityToID(v view.Identity) (view.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping view.Identity identifier for [%s], default identities [%s:%s]",
			r.networkID,
			v,
			defaultID.String(),
			r.nodeIdentity.String(),
		)
	}

	id := v
	switch {
	case id.IsNone():
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed empty identity")
		}
		return nil, defaultIdentifier, nil
	case id.Equal(defaultID):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed default identity")
		}
		return nil, defaultIdentifier, nil
	case string(id) == defaultIdentifier:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed 'idemix' identity")
		}
		return nil, defaultIdentifier, nil
	case id.Equal(r.nodeIdentity):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed identity is the node identity (same bytes)")
		}
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(id):
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("passed identity is me")
		}
		return id, "", nil
	}
	label := string(id)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("looking up identifier for identity as label [%s]", hash.Hashable(label))
	}

	if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
		return nil, idIdentifier, nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("cannot find match for view.Identity string [%s]", id)
	}
	return nil, string(id), nil
}

// RegisterIdentity registers the given identity
func (r *Role) RegisterIdentity(config driver.IdentityConfiguration) error {
	return r.localMembership.RegisterIdentity(config)
}

func (r *Role) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}
