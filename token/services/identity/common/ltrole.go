/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// LongTermRole models a role whose identities are not anonymous
type LongTermRole struct {
	*Role
	nodeIdentity driver.Identity
}

func NewLongTermRole(logger logging.Logger, roleID identity.RoleType, networkID string, nodeIdentity driver.Identity, localMembership localMembership) *LongTermRole {
	return &LongTermRole{
		Role:         NewRole(logger, roleID, networkID, localMembership),
		nodeIdentity: nodeIdentity,
	}
}

// MapToIdentity returns the identity for the given argument
func (r *LongTermRole) MapToIdentity(v driver.WalletLookupID) (driver.Identity, string, error) {
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

func (r *LongTermRole) mapStringToID(v string) (driver.Identity, string, error) {
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

func (r *LongTermRole) mapIdentityToID(v driver.Identity) (driver.Identity, string, error) {
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
		return nil, idIdentifier, nil
	}

	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("cannot find match for driver.Identity string [%s]", id)
	}
	return nil, string(id), nil
}
