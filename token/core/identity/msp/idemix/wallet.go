/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"go.uber.org/zap/zapcore"
)

type localMembership interface {
	DefaultNetworkIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error)
	GetIdentifier(id view.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(id string, path string) error
}

// wallet maps an identifier to an identity
type wallet struct {
	networkID       string
	nodeIdentity    view.Identity
	localMembership localMembership
}

func NewWallet(networkID string, nodeIdentity view.Identity, localMembership localMembership) *wallet {
	return &wallet{
		networkID:       networkID,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (w *wallet) GetIdentityInfo(id string) driver.IdentityInfo {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] getting info for [%s]", w.networkID, id)
	}

	info, err := w.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[%s] failed to get anonymous identity for [%s]: %s", w.networkID, id, err)
		}
		return nil
	}
	return info
}

// MapToID returns the identity for the given argument
func (w *wallet) MapToID(v interface{}) (view.Identity, string) {
	defaultID := w.localMembership.DefaultNetworkIdentity()
	defaultIdentifier := w.localMembership.GetDefaultIdentifier()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping identifier for [%d,%s], default identities [%s:%s,%s]",
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
			logger.Debugf("[AnonymousIdentity] looking up identifier for identity [%d,%s]", vv.String())
		}
		id := vv
		switch {
		case id.IsNone():
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed empty identity")
			}
			return nil, defaultIdentifier
		case id.Equal(defaultID):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identity")
			}
			return nil, defaultIdentifier
		case string(id) == defaultIdentifier:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed 'idemix' identity")
			}
			return nil, defaultIdentifier
		case id.Equal(w.nodeIdentity):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed identity is the node identity (same bytes)")
			}
			return nil, defaultIdentifier
		case w.localMembership.IsMe(id):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed identity is me")
			}
			return id, ""
		}
		label := string(id)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] looking up identifier for identity as label [%d,%s]", label)
		}

		if idIdentifier, err := w.localMembership.GetIdentifier(id); err == nil {
			return nil, idIdentifier
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] cannot find match for view.Identity string [%s]", vv)
		}
		return nil, string(id)
	case string:
		label := vv
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] looking up identifier for label [%d,%s]", vv)
		}
		switch {
		case len(label) == 0:
			return nil, defaultIdentifier
		case label == defaultIdentifier:
			return nil, defaultIdentifier
		case label == defaultID.UniqueID():
			return nil, defaultIdentifier
		case label == string(defaultID):
			return nil, defaultIdentifier
		case defaultID.Equal(view.Identity(label)):
			return nil, defaultIdentifier
		case w.nodeIdentity.Equal(view.Identity(label)):
			return nil, defaultIdentifier
		case w.localMembership.IsMe(view.Identity(label)):
			return nil, defaultIdentifier
		}

		if idIdentifier, err := w.localMembership.GetIdentifier(view.Identity(label)); err == nil {
			return nil, idIdentifier
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] cannot find match for view.Identity string [%s]", vv)
		}
		return nil, label
	default:
		panic("[AnonymousIdentity] identifier not recognised, expected []byte or view.Identity")
	}
}

// RegisterIdentity registers the given identity
func (w *wallet) RegisterIdentity(id string, path string) error {
	return w.localMembership.RegisterIdentity(id, path)
}
