/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"go.uber.org/zap/zapcore"
)

type LocalMembership interface {
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
	localMembership LocalMembership
}

func NewMapper(networkID string, nodeIdentity view.Identity, localMembership LocalMembership) *mapper {
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
			logger.Debugf("[%s] failed to get long term identity for [%s]: %s", i.networkID, id, err)
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
			logger.Debugf(
				"[LongTermIdentity] looking up identifier for identity [%d,%s], default identity [%s]",
				vv.String(),
				defaultID.String(),
			)
		}
		id := vv
		switch {
		case id.IsNone():
			return defaultID, DefaultLabel
		case id.Equal(defaultID):
			return defaultID, DefaultLabel
		case id.Equal(i.nodeIdentity):
			return defaultID, DefaultLabel
		case i.localMembership.IsMe(id):
			if idIdentifier, err := i.localMembership.GetIdentifier(id); err == nil {
				return id, idIdentifier
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed getting identity info for [%s], returning the identity", id)
			}
			return id, ""
		case string(id) == DefaultLabel:
			return defaultID, DefaultLabel
		}

		label := string(id)
		if info, err := i.localMembership.GetIdentityInfo(label, nil); err == nil {
			id, _, err := info.Get()
			if err != nil {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("failed getting identity info for [%s], returning the identity", id)
				}
				return nil, info.ID()
			}
			return id, label
		}
		if idIdentifier, err := i.localMembership.GetIdentifier(id); err == nil {
			return id, idIdentifier
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", vv)
		}

		return id, ""
	case string:
		label := vv
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[LongTermIdentity] looking up identifier for label [%d,%s]", vv)
		}
		switch {
		case len(label) == 0:
			return defaultID, DefaultLabel
		case label == DefaultLabel:
			return defaultID, DefaultLabel
		case label == defaultID.UniqueID():
			return defaultID, DefaultLabel
		case label == string(defaultID):
			return defaultID, DefaultLabel
		case defaultID.Equal(view.Identity(label)):
			return defaultID, DefaultLabel
		case i.nodeIdentity.Equal(view.Identity(label)):
			return defaultID, DefaultLabel
		case i.localMembership.IsMe(view.Identity(label)):
			id := view.Identity(label)
			if idIdentifier, err := i.localMembership.GetIdentifier(id); err == nil {
				return id, idIdentifier
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[LongTermIdentity] failed getting identity info for [%s], returning the identity", id)
			}
			return id, ""
		}

		if info, err := i.localMembership.GetIdentityInfo(label, nil); err == nil {
			id, _, err := info.Get()
			if err != nil {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("failed getting identity info for [%s], returning the identity", id)
				}
				return nil, info.ID()
			}
			return id, label
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", vv)
		}
		return nil, label
	default:
		panic(fmt.Sprintf("[LongTermIdentity] identifier not recognised, expected []byte or view.Identity"))
	}
}

func (i *mapper) RegisterIdentity(id string, typ string, path string) error {
	return i.localMembership.RegisterIdentity(id, typ, path)
}
