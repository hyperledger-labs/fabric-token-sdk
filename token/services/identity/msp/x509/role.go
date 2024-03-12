/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
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
	networkID       string
	nodeIdentity    view.Identity
	localMembership localMembership
}

func NewRole(networkID string, nodeIdentity view.Identity, localMembership localMembership) *Role {
	return &Role{
		networkID:       networkID,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (w *Role) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] getting info for [%s]", w.networkID, id)
	}

	info, err := w.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[%s] failed to get long term identity for [%s]: %s", w.networkID, id, err)
		}
		return nil, nil
	}
	return info, nil
}

// MapToID returns the identity for the given argument
func (w *Role) MapToID(v interface{}) (view.Identity, string, error) {
	defaultID := w.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := w.localMembership.GetDefaultIdentifier()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping identifier for [%s,%s], default identities [%s:%s]",
			w.networkID,
			v,
			string(defaultID),
			defaultID.String(),
			w.nodeIdentity.String(),
		)
	}

	switch vv := v.(type) {
	case view.Identity:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf(
				"[LongTermIdentity] looking up identifier for identity [%s], default identity [%s]",
				vv.String(),
				defaultID.String(),
			)
		}
		id := vv
		switch {
		case id.IsNone():
			return defaultID, defaultIdentifier, nil
		case id.Equal(defaultID):
			return defaultID, defaultIdentifier, nil
		case id.Equal(w.nodeIdentity):
			return defaultID, defaultIdentifier, nil
		case w.localMembership.IsMe(id):
			if idIdentifier, err := w.localMembership.GetIdentifier(id); err == nil {
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
		if info, err := w.localMembership.GetIdentityInfo(label, nil); err == nil {
			id, _, err := info.Get()
			if err != nil {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("failed getting identity info for [%s], returning the identity", id)
				}
				return nil, info.ID(), nil
			}
			return id, label, nil
		}
		if idIdentifier, err := w.localMembership.GetIdentifier(id); err == nil {
			return id, idIdentifier, nil
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", vv)
		}

		return id, "", nil
	case string:
		label := vv
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[LongTermIdentity] looking up identifier for label [%s]", vv)
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
		case w.nodeIdentity.Equal(view.Identity(label)):
			return defaultID, defaultIdentifier, nil
		case w.localMembership.IsMe(view.Identity(label)):
			id := view.Identity(label)
			if idIdentifier, err := w.localMembership.GetIdentifier(id); err == nil {
				return id, idIdentifier, nil
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[LongTermIdentity] failed getting identity info for [%s], returning the identity", id)
			}
			return id, "", nil
		}

		if info, err := w.localMembership.GetIdentityInfo(label, nil); err == nil {
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
			logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", vv)
		}
		return nil, label, nil
	default:
		return nil, "", errors.Errorf("[LongTermIdentity] identifier not recognised, expected []byte or view.Identity")
	}
}

// RegisterIdentity registers the given identity
func (w *Role) RegisterIdentity(id string, path string) error {
	logger.Debugf("register x509 identity [%s:%s]", id, path)
	return w.localMembership.RegisterIdentity(id, path)
}

func (w *Role) IDs() ([]string, error) {
	return w.localMembership.IDs()
}

func (w *Role) Reload(pp driver.PublicParameters) error {
	logger.Debugf("reload x509 wallets...")
	// nothing to do here
	return nil
}
