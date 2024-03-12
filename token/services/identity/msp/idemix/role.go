/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
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
	RegisterIdentity(id string, path string) error
	IDs() ([]string, error)
	Reload(pp driver.PublicParameters) error
}

// Role is a container of idemix-based long-term identities.
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
			logger.Debugf("[%s] failed to get anonymous identity for [%s]: %s", w.networkID, id, err)
		}
		return nil, nil
	}
	return info, nil
}

// MapToID returns the identity for the given argument
func (w *Role) MapToID(v interface{}) (view.Identity, string, error) {
	defaultID := w.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := w.localMembership.GetDefaultIdentifier()

	switch vv := v.(type) {
	case view.Identity:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] [%s] mapping view.Identity identifier for [%s,%s], default identities [%s:%s]",
				w.networkID,
				v,
				vv.String(),
				defaultID.String(),
				w.nodeIdentity.String(),
			)
		}

		id := vv
		switch {
		case id.IsNone():
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed empty identity")
			}
			return nil, defaultIdentifier, nil
		case id.Equal(defaultID):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identity")
			}
			return nil, defaultIdentifier, nil
		case string(id) == defaultIdentifier:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed 'idemix' identity")
			}
			return nil, defaultIdentifier, nil
		case id.Equal(w.nodeIdentity):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed identity is the node identity (same bytes)")
			}
			return nil, defaultIdentifier, nil
		case w.localMembership.IsMe(id):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed identity is me")
			}
			return id, "", nil
		}
		label := string(id)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] looking up identifier for identity as label [%s]", hash.Hashable(label))
		}

		if idIdentifier, err := w.localMembership.GetIdentifier(id); err == nil {
			return nil, idIdentifier, nil
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] cannot find match for view.Identity string [%s]", hash.Hashable(vv).String())
		}
		return nil, string(id), nil
	case string:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] [%s] mapping string identifier for [%s,%s], default identities [%s:%s]",
				w.networkID,
				v,
				hash.Hashable(vv).String(),
				defaultID.String(),
				w.nodeIdentity.String(),
			)
		}

		label := vv
		viewIdentity := view.Identity(label)
		switch {
		case len(label) == 0:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed empty identity")
			}
			return nil, defaultIdentifier, nil
		case label == defaultIdentifier:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identifier")
			}
			return nil, defaultIdentifier, nil
		case label == defaultID.UniqueID():
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identity")
			}
			return nil, defaultIdentifier, nil
		case label == string(defaultID):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identity as string")
			}
			return nil, defaultIdentifier, nil
		case defaultID.Equal(viewIdentity):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identity as view identity")
			}
			return nil, defaultIdentifier, nil
		case w.nodeIdentity.Equal(viewIdentity):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed node identity as view identity")
			}
			return nil, defaultIdentifier, nil
		case w.localMembership.IsMe(viewIdentity):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed a local member")
			}
			return nil, defaultIdentifier, nil
		}

		if idIdentifier, err := w.localMembership.GetIdentifier(viewIdentity); err == nil {
			return nil, idIdentifier, nil
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] cannot find match for string [%s]", vv)
		}
		return nil, label, nil
	default:
		return nil, "", errors.Errorf("[AnonymousIdentity] identifier not recognised, expected []byte or view.Identity")
	}
}

// RegisterIdentity registers the given identity
func (w *Role) RegisterIdentity(id string, path string) error {
	logger.Debugf("register idemix identity [%s:%s]", id, path)
	return w.localMembership.RegisterIdentity(id, path)
}

func (w *Role) IDs() ([]string, error) {
	return w.localMembership.IDs()
}

func (w *Role) Reload(pp driver.PublicParameters) error {
	logger.Debugf("reload idemix wallets...")
	return w.localMembership.Reload(pp)
}
