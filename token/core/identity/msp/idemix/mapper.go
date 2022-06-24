/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"go.uber.org/zap/zapcore"
)

type localMembership interface {
	DefaultIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error)
	GetIdentifier(id view.Identity) (string, error)
	RegisterIdentity(id string, typ string, path string) error
}

// mapper maps identifiers of different sorts to identities
type mapper struct {
	networkID       string
	nodeIdentity    view.Identity
	localMembership localMembership
}

func NewMapper(networkID string, nodeIdentity view.Identity, localMembership localMembership) *mapper {
	return &mapper{
		networkID:       networkID,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

// GetIdentityInfo get in input an identifier and returns:
// - The corresponding long term identifier
// - The corresponding enrollment ID
// - A function that returns the identity and its audit info.
func (i *mapper) GetIdentityInfo(id string) driver.IdentityInfo {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] getting info for [%s]", i.networkID, id)
	}

	info, err := i.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[%s] failed to get anonymous identity for [%s]: %s", i.networkID, id, err)
		}
		return nil
	}
	return info
}

func (i *mapper) MapToID(v interface{}) (view.Identity, string) {
	defaultID := i.localMembership.DefaultIdentity()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping identifier for [%d,%s], default identities [%s:%s,%s]",
			i.networkID,
			v,
			string(defaultID),
			defaultID.String(),
			i.nodeIdentity.String(),
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
			return nil, DefaultLabel
		case id.Equal(defaultID):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed default identity")
			}
			return nil, DefaultLabel
		case string(id) == DefaultLabel:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed 'idemix' identity")
			}
			return nil, DefaultLabel
		case id.Equal(i.nodeIdentity):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed identity is the node identity (same bytes)")
			}
			return nil, DefaultLabel
		case i.localMembership.IsMe(id):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] passed identity is me")
			}
			return id, ""
		}
		label := string(id)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] looking up identifier for identity as label [%d,%s]", label)
		}

		if idIdentifier, err := i.localMembership.GetIdentifier(id); err == nil {
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
			return nil, DefaultLabel
		case label == DefaultLabel:
			return nil, DefaultLabel
		case label == defaultID.UniqueID():
			return nil, DefaultLabel
		case label == string(defaultID):
			return nil, DefaultLabel
		case defaultID.Equal(view.Identity(label)):
			return nil, DefaultLabel
		case i.nodeIdentity.Equal(view.Identity(label)):
			return nil, DefaultLabel
		case i.localMembership.IsMe(view.Identity(label)):
			return nil, DefaultLabel
		}

		if idIdentifier, err := i.localMembership.GetIdentifier(view.Identity(label)); err == nil {
			return nil, idIdentifier
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[AnonymousIdentity] cannot find match for view.Identity string [%s]", vv)
		}
		return nil, label
	default:
		panic(fmt.Sprintf("[AnonymousIdentity] identifier not recognised, expected []byte or view.Identity"))
	}
}

func (i *mapper) RegisterIdentity(id string, typ string, path string) error {
	return i.localMembership.RegisterIdentity(id, typ, path)
}
