/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type localMembership interface {
	DefaultNetworkIdentity() driver.Identity
	IsMe(id driver.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error)
	GetIdentifier(id driver.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(config driver.IdentityConfiguration) error
	IDs() ([]string, error)
}

// AnonymousRole models a role whose identities are anonymous
type AnonymousRole struct {
	logger          logging.Logger
	roleID          driver.IdentityRole
	networkID       string
	nodeIdentity    driver.Identity
	localMembership localMembership
}

func NewAnonymousRole(logger logging.Logger, roleID driver.IdentityRole, networkID string, nodeIdentity driver.Identity, localMembership localMembership) *AnonymousRole {
	return &AnonymousRole{
		logger:          logger,
		roleID:          roleID,
		networkID:       networkID,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

func (r *AnonymousRole) ID() driver.IdentityRole {
	return r.roleID
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (r *AnonymousRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] getting info for [%s]", r.networkID, id)
	}

	info, err := r.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "[%s] failed to get identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// MapToID returns the identity for the given argument
func (r *AnonymousRole) MapToID(v driver.WalletLookupID) (driver.Identity, string, error) {
	switch vv := v.(type) {
	case []byte:
		return r.mapIdentityToID(vv)
	case driver.Identity:
		return r.mapIdentityToID(vv)
	case string:
		return r.mapStringToID(vv)
	default:
		return nil, "", errors.Errorf("identifier not recognised, expected []byte or driver.Identity, got [%T], [%s]", v, string(debug.Stack()))
	}
}

// RegisterIdentity registers the given identity
func (r *AnonymousRole) RegisterIdentity(config driver.IdentityConfiguration) error {
	return r.localMembership.RegisterIdentity(config)
}

func (r *AnonymousRole) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}

func (r *AnonymousRole) mapStringToID(v string) (driver.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] mapping string identifier for [%s,%s], default identities [%s:%s]",
			r.networkID,
			v,
			hash.Hashable(v).String(),
			defaultID.String(),
			r.nodeIdentity.String(),
		)
	}

	label := v
	viewIdentity := driver.Identity(label)
	switch {
	case len(label) == 0:
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed empty identity")
		}
		return nil, defaultIdentifier, nil
	case label == defaultIdentifier:
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed default identifier")
		}
		return nil, defaultIdentifier, nil
	case label == defaultID.UniqueID():
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed default identity")
		}
		return nil, defaultIdentifier, nil
	case label == string(defaultID):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed default identity as string")
		}
		return nil, defaultIdentifier, nil
	case defaultID.Equal(viewIdentity):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed default identity as view identity")
		}
		return nil, defaultIdentifier, nil
	case r.nodeIdentity.Equal(viewIdentity):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed node identity as view identity")
		}
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(viewIdentity):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed a local member")
		}
		return nil, defaultIdentifier, nil
	}

	if idIdentifier, err := r.localMembership.GetIdentifier(viewIdentity); err == nil {
		return nil, idIdentifier, nil
	}
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("cannot find match for string [%s]", v)
	}
	return nil, label, nil
}

func (r *AnonymousRole) mapIdentityToID(v driver.Identity) (driver.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] mapping driver.Identity identifier for [%s], default identities [%s:%s]",
			r.networkID,
			v,
			defaultID.String(),
			r.nodeIdentity.String(),
		)
	}

	id := v
	switch {
	case id.IsNone():
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed empty identity")
		}
		return nil, defaultIdentifier, nil
	case id.Equal(defaultID):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed default identity")
		}
		return nil, defaultIdentifier, nil
	case string(id) == defaultIdentifier:
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed 'idemix' identity")
		}
		return nil, defaultIdentifier, nil
	case id.Equal(r.nodeIdentity):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed identity is the node identity (same bytes)")
		}
		return nil, defaultIdentifier, nil
	case r.localMembership.IsMe(id):
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("passed identity is me")
		}
		return id, "", nil
	}
	label := string(id)
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("looking up identifier for identity as label [%s]", hash.Hashable(label))
	}

	if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
		return nil, idIdentifier, nil
	}
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("cannot find match for driver.Identity string [%s]", id)
	}
	return nil, string(id), nil
}
