/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
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

// Role models a generic role
type Role struct {
	logger          logging.Logger
	roleID          driver.IdentityRole
	networkID       string
	localMembership localMembership
}

func NewRole(logger logging.Logger, roleID driver.IdentityRole, networkID string, localMembership localMembership) *Role {
	return &Role{
		logger:          logger,
		roleID:          roleID,
		networkID:       networkID,
		localMembership: localMembership,
	}
}

func (r *Role) ID() driver.IdentityRole {
	return r.roleID
}

// GetIdentityInfo returns the identity information for the given identity identifier
func (r *Role) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	if r.logger.IsEnabledFor(zapcore.DebugLevel) {
		r.logger.Debugf("[%s] getting info for [%s]", r.networkID, id)
	}

	info, err := r.localMembership.GetIdentityInfo(id, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "[%s] failed to get identity for [%s]", r.networkID, id)
	}
	return info, nil
}

// RegisterIdentity registers the given identity
func (r *Role) RegisterIdentity(config driver.IdentityConfiguration) error {
	return r.localMembership.RegisterIdentity(config)
}

func (r *Role) IdentityIDs() ([]string, error) {
	return r.localMembership.IDs()
}
