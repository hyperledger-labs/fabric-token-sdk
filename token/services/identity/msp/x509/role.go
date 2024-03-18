/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"runtime/debug"

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
	RegisterIdentity(id string, path string) error
	IDs() ([]string, error)
}

// Role is a container of x509-based long-term identities.
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
		return nil, errors.WithMessagef(err, "[%s] failed to get long term identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// MapToID returns the identity for the given argument
func (r *Role) MapToID(v driver.WalletLookupID) (view.Identity, string, error) {
	switch vv := v.(type) {
	case view.Identity:
		return r.mapIdentityToID(vv)
	case []byte:
		return r.mapIdentityToID(vv)
	case string:
		return r.mapStringToID(vv)
	default:
		return nil, "", errors.Errorf("[LongTermIdentity] identifier not recognised, expected []byte or view.Identity, got [%T], [%s]", v, debug.Stack())
	}
}

func (r *Role) mapStringToID(v string) (view.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping identifier for [%s,%s], default identities [%s:%s]",
			r.networkID,
			v,
			string(defaultID),
			defaultID.String(),
			r.nodeIdentity.String(),
		)
	}

	label := v
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[LongTermIdentity] looking up identifier for label [%s]", label)
	}
	switch {
	case len(label) == 0:
		return defaultID, defaultIdentifier, nil
	case label == defaultIdentifier:
		return defaultID, defaultIdentifier, nil
	case label == defaultID.UniqueID():
		return defaultID, defaultIdentifier, nil
	case label == string(defaultID):
		return defaultID, defaultIdentifier, nil
	case defaultID.Equal(view.Identity(label)):
		return defaultID, defaultIdentifier, nil
	case r.nodeIdentity.Equal(view.Identity(label)):
		return defaultID, defaultIdentifier, nil
	case r.localMembership.IsMe(view.Identity(label)):
		id := view.Identity(label)
		if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
			return id, idIdentifier, nil
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[LongTermIdentity] failed getting identity info for [%s], returning the identity", id)
		}
		return id, "", nil
	}

	if info, err := r.localMembership.GetIdentityInfo(label, nil); err == nil {
		id, _, err := info.Get()
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed getting identity info for [%s], returning the identity", id)
			}
			return nil, info.ID(), nil
		}
		return id, label, nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", label)
	}
	return nil, label, nil
}

func (r *Role) mapIdentityToID(v view.Identity) (view.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf(
			"[LongTermIdentity] looking up identifier for identity [%s], default identity [%s]",
			v,
			defaultID.String(),
		)
	}
	id := v
	switch {
	case id.IsNone():
		return defaultID, defaultIdentifier, nil
	case id.Equal(defaultID):
		return defaultID, defaultIdentifier, nil
	case id.Equal(r.nodeIdentity):
		return defaultID, defaultIdentifier, nil
	case r.localMembership.IsMe(id):
		if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
			return id, idIdentifier, nil
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting identity info for [%s], returning the identity", id)
		}
		return id, "", nil
	case string(id) == defaultIdentifier:
		return defaultID, defaultIdentifier, nil
	}

	label := string(id)
	if info, err := r.localMembership.GetIdentityInfo(label, nil); err == nil {
		id, _, err := info.Get()
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed getting identity info for [%s], returning the identity", id)
			}
			return nil, info.ID(), nil
		}
		return id, label, nil
	}
	if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
		return id, idIdentifier, nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", v)
	}

	return id, "", nil
}

// RegisterIdentity registers the given identity
func (r *Role) RegisterIdentity(id string, path string) error {
	logger.Debugf("register x509 identity [%s:%s]", id, path)
	return r.localMembership.RegisterIdentity(id, path)
}

func (r *Role) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}

func (r *Role) Reload(pp driver.PublicParameters) error {
	logger.Debugf("reload x509 wallets...")
	// nothing to do here
	return nil
}
