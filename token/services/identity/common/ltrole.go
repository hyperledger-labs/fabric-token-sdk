/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// LongTermRole models a role whose identities are not anonymous
type LongTermRole struct {
	logger          logging.Logger
	roleID          driver.IdentityRole
	networkID       string
	nodeIdentity    driver.Identity
	localMembership localMembership
}

func NewLongTermRole(logger logging.Logger, roleID driver.IdentityRole, networkID string, nodeIdentity driver.Identity, localMembership localMembership) *LongTermRole {
	return &LongTermRole{
		logger:          logger,
		roleID:          roleID,
		networkID:       networkID,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

func (r *LongTermRole) ID() driver.IdentityRole {
	return r.roleID
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (r *LongTermRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] getting info for [%s]", r.networkID, id)
	}

	info, err := r.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "[%s] failed to get long term identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// MapToID returns the identity for the given argument
func (r *LongTermRole) MapToID(v driver.WalletLookupID) (driver.Identity, string, error) {
	switch vv := v.(type) {
	case driver.Identity:
		return r.mapIdentityToID(vv)
	case []byte:
		return r.mapIdentityToID(vv)
	case string:
		return r.mapStringToID(vv)
	default:
		return nil, "", errors.Errorf("[LongTermIdentity] identifier not recognised, expected []byte or driver.Identity, got [%T], [%s]", v, debug.Stack())
	}
}

func (r *LongTermRole) mapStringToID(v string) (driver.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] mapping identifier for [%s,%s], default identities [%s:%s]",
			r.networkID,
			v,
			string(defaultID),
			defaultID.String(),
			r.nodeIdentity.String(),
		)
	}

	label := v
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[LongTermIdentity] looking up identifier for label [%s]", label)
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
	case defaultID.Equal(driver.Identity(label)):
		return defaultID, defaultIdentifier, nil
	case r.nodeIdentity.Equal(driver.Identity(label)):
		return defaultID, defaultIdentifier, nil
	case r.localMembership.IsMe(driver.Identity(label)):
		id := driver.Identity(label)
		if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
			return id, idIdentifier, nil
		}
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("[LongTermIdentity] failed getting identity info for [%s], returning the identity", id)
		}
		return id, "", nil
	}

	if info, err := r.localMembership.GetIdentityInfo(label, nil); err == nil {
		id, _, err := info.Get()
		if err != nil {
			if r.logger.IsEnabledFor(zapcore.DebugLevel) {
				r.logger.Debugf("failed getting identity info for [%s], returning the identity", id)
			}
			return nil, info.ID(), nil
		}
		return id, label, nil
	}
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[LongTermIdentity] cannot find match for driver.Identity string [%s]", label)
	}
	return nil, label, nil
}

func (r *LongTermRole) mapIdentityToID(v driver.Identity) (driver.Identity, string, error) {
	defaultID := r.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := r.localMembership.GetDefaultIdentifier()

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf(
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
		if r.logger.IsEnabledFor(zapcore.DebugLevel) {
			r.logger.Debugf("failed getting identity info for [%s], returning the identity", id)
		}
		return id, "", nil
	case string(id) == defaultIdentifier:
		return defaultID, defaultIdentifier, nil
	}

	label := string(id)
	if info, err := r.localMembership.GetIdentityInfo(label, nil); err == nil {
		id, _, err := info.Get()
		if err != nil {
			if r.logger.IsEnabledFor(zapcore.DebugLevel) {
				r.logger.Debugf("failed getting identity info for [%s], returning the identity", id)
			}
			return nil, info.ID(), nil
		}
		return id, label, nil
	}
	if idIdentifier, err := r.localMembership.GetIdentifier(id); err == nil {
		return id, idIdentifier, nil
	}
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[LongTermIdentity] cannot find match for driver.Identity string [%s]", v)
	}

	return id, "", nil
}

// RegisterIdentity registers the given identity
func (r *LongTermRole) RegisterIdentity(config driver.IdentityConfiguration) error {
	return r.localMembership.RegisterIdentity(config)
}

func (r *LongTermRole) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}
