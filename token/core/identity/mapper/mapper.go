/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mapper

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity.tms")

type IdentityType int

const (
	LongTermIdentity IdentityType = iota
	AnonymousIdentity
)

type LocalMembership interface {
	DefaultIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetAnonymousIdentity(label string, auditInfo []byte) (string, string, network.GetFunc, error)
	GetAnonymousIdentifier(label string) (string, error)
	GetLongTermIdentity(label string) (string, string, view.Identity, error)
	GetLongTermIdentifier(id view.Identity) (string, error)
	RegisterIdentity(id string, typ string, path string) error
}

// mapper maps identifiers of different sorts to identities
type mapper struct {
	networkID       string
	nodeIdentity    view.Identity
	localMembership LocalMembership
	identityType    IdentityType
}

func NewMapper(networkID string, identityType IdentityType, nodeIdentity view.Identity, localMembership LocalMembership) *mapper {
	return &mapper{
		networkID:       networkID,
		identityType:    identityType,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

// Info get in input an identifier and returns:
// - The corresponding long term identifier
// - The corresponding enrollment ID
// - A function that returns the identity and its audit info.
func (i *mapper) Info(id string) (string, string, GetFunc) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] getting info for [%s]", i.networkID, id)
	}

	switch i.identityType {
	case LongTermIdentity:
		id, eID, longTermID, err := i.localMembership.GetLongTermIdentity(id)
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[%s] failed to get long term identity for [%s]: %s", i.networkID, id, err)
			}
			return "", "", nil
		}
		return id, eID, func() (view.Identity, []byte, error) {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[%s] return [%s][%s][%s]", i.networkID, id, longTermID, eID)
			}
			return longTermID, []byte(eID), nil
		}
	case AnonymousIdentity:
		id, eID, getFunc, err := i.localMembership.GetAnonymousIdentity(id, nil)
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[%s] failed to get anonymous identity for [%s]: %s", i.networkID, id, err)
			}
			return "", "", nil
		}
		return id, eID, GetFunc(getFunc)
	default:
		panic(fmt.Sprintf("type not recognized [%d]", i.identityType))
	}
}

func (i *mapper) Map(v interface{}) (view.Identity, string) {
	defaultID := i.localMembership.DefaultIdentity()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] mapping identifier for [%d,%s], default identities [%s:%s,%s]",
			i.networkID,
			i.identityType,
			v,
			string(defaultID),
			defaultID.String(),
			i.nodeIdentity.String(),
		)
	}

	switch i.identityType {
	case LongTermIdentity:
		switch vv := v.(type) {
		case view.Identity:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf(
					"[LongTermIdentity] looking up identifier for identity [%d,%s], default identity [%s]",
					i.identityType,
					vv.String(),
					defaultID.String(),
				)
			}
			id := vv
			switch {
			case id.IsNone():
				return defaultID, "default"
			case id.Equal(defaultID):
				return defaultID, "default"
			case id.Equal(i.nodeIdentity):
				return defaultID, "default"
			case i.localMembership.IsMe(id):
				if idIdentifier, err := i.localMembership.GetLongTermIdentifier(id); err == nil {
					return id, idIdentifier
				}
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("failed getting identity info for [%s], returning the identity", id)
				}
				return id, ""
			case string(id) == "default":
				return defaultID, "default"
			}

			label := string(id)
			if _, _, longTermID, err := i.localMembership.GetLongTermIdentity(label); err == nil {
				return longTermID, label
			}
			if idIdentifier, err := i.localMembership.GetLongTermIdentifier(id); err == nil {
				return id, idIdentifier
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", vv)
			}

			return id, ""
		case string:
			label := vv
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[LongTermIdentity] looking up identifier for label [%d,%s]", i.identityType, vv)
			}
			switch {
			case len(label) == 0:
				return defaultID, "default"
			case label == "default":
				return defaultID, "default"
			case label == defaultID.UniqueID():
				return defaultID, "default"
			case label == string(defaultID):
				return defaultID, "default"
			case defaultID.Equal(view.Identity(label)):
				return defaultID, "default"
			case i.nodeIdentity.Equal(view.Identity(label)):
				return defaultID, "default"
			case i.localMembership.IsMe(view.Identity(label)):
				id := view.Identity(label)
				if idIdentifier, err := i.localMembership.GetLongTermIdentifier(id); err == nil {
					return id, idIdentifier
				}
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[LongTermIdentity] failed getting identity info for [%s], returning the identity", id)
				}
				return id, ""
			}

			if _, _, longTermID, err := i.localMembership.GetLongTermIdentity(label); err == nil {
				return longTermID, label
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[LongTermIdentity] cannot find match for view.Identity string [%s]", vv)
			}
			return nil, label
		default:
			panic(fmt.Sprintf("[LongTermIdentity] identifier not recognised, expected []byte or view.Identity"))
		}
	case AnonymousIdentity:
		switch vv := v.(type) {
		case view.Identity:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] looking up identifier for identity [%d,%s]", i.identityType, vv.String())
			}
			id := vv
			switch {
			case id.IsNone():
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[AnonymousIdentity] passed empty identity")
				}
				return nil, "idemix"
			case id.Equal(defaultID):
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[AnonymousIdentity] passed default identity")
				}
				return nil, "idemix"
			case string(id) == "idemix":
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[AnonymousIdentity] passed 'idemix' identity")
				}
				return nil, "idemix"
			case id.Equal(i.nodeIdentity):
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[AnonymousIdentity] passed identity is the node identity (same bytes)")
				}
				return nil, "idemix"
			case i.localMembership.IsMe(id):
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[AnonymousIdentity] passed identity is me")
				}
				return id, ""
			}
			label := string(id)
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] looking up identifier for identity as label [%d,%s]", i.identityType, label)
			}

			if idIdentifier, err := i.localMembership.GetAnonymousIdentifier(label); err == nil {
				return nil, idIdentifier
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] cannot find match for view.Identity string [%s]", vv)
			}
			return nil, string(id)
		case string:
			label := vv
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] looking up identifier for label [%d,%s]", i.identityType, vv)
			}
			switch {
			case len(label) == 0:
				return nil, "idemix"
			case label == "idemix":
				return nil, "idemix"
			case label == defaultID.UniqueID():
				return nil, "idemix"
			case label == string(defaultID):
				return nil, "idemix"
			case defaultID.Equal(view.Identity(label)):
				return nil, "idemix"
			case i.nodeIdentity.Equal(view.Identity(label)):
				return nil, "idemix"
			case i.localMembership.IsMe(view.Identity(label)):
				return nil, "idemix"
			}

			if idIdentifier, err := i.localMembership.GetAnonymousIdentifier(label); err == nil {
				return nil, idIdentifier
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[AnonymousIdentity] cannot find match for view.Identity string [%s]", vv)
			}
			return nil, label
		default:
			panic(fmt.Sprintf("[AnonymousIdentity] identifier not recognised, expected []byte or view.Identity"))
		}
	default:
		panic(fmt.Sprintf("msp type [%d] not supported", i.identityType))
	}
}

func (i *mapper) RegisterIdentity(id string, typ string, path string) error {
	return i.localMembership.RegisterIdentity(id, typ, path)
}
