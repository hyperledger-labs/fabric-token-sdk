/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	"fmt"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity.fabric")

type MSPType int

const (
	X509MSPIdentity MSPType = iota
	IdemixMSPIdentity
)

const (
	IdemixMSP = "idemix"
	BccspMSP  = "bccsp"
)

//go:generate counterfeiter -o mock/local_membership.go -fake-name LocalMembership . LocalMembership

type LocalMembership interface {
	DefaultIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetIdentityInfoByLabel(mspType string, label string) *fabric.IdentityInfo
	GetIdentityInfoByIdentity(mspType string, id view.Identity) *fabric.IdentityInfo
}

type Mapper struct {
	networkID       string
	nodeIdentity    view.Identity
	localMembership LocalMembership
	mspType         MSPType
}

func NewMapper(networkID string, mspType MSPType, nodeIdentity view.Identity, localMembership LocalMembership) *Mapper {
	return &Mapper{
		networkID:       networkID,
		mspType:         mspType,
		nodeIdentity:    nodeIdentity,
		localMembership: localMembership,
	}
}

func (i *Mapper) Info(id string) (string, string, identity.GetFunc) {
	logger.Debugf("[%s] getting info for [%s]", i.networkID)

	var mspLabel string
	switch i.mspType {
	case X509MSPIdentity:
		mspLabel = BccspMSP
	case IdemixMSPIdentity:
		mspLabel = IdemixMSP
	default:
		panic(fmt.Sprintf("type not recognized [%d]", i.mspType))
	}
	idInfo := i.localMembership.GetIdentityInfoByLabel(mspLabel, id)
	if idInfo == nil {
		return "", "", nil
	}
	return idInfo.ID, idInfo.EnrollmentID, func() (view.Identity, []byte, error) {
		return idInfo.GetIdentity()
	}
}

func (i *Mapper) Map(v interface{}) (view.Identity, string) {
	defaultID := i.localMembership.DefaultIdentity()

	logger.Debugf("[%s] mapping identifier for [%d,%s], default identities [%s:%s,%s] [%s]",
		i.networkID,
		i.mspType,
		v,
		string(defaultID),
		defaultID.String(),
		i.nodeIdentity.String(),
		debug.Stack(),
	)

	switch i.mspType {
	case X509MSPIdentity:
		switch vv := v.(type) {
		case view.Identity:
			logger.Debugf(
				"[x509] looking up identifier for identity [%d,%s], default identity [%s]",
				i.mspType,
				vv.String(),
				defaultID.String(),
			)
			id := vv
			switch {
			case id.IsNone():
				return defaultID, "default"
			case id.Equal(defaultID):
				return defaultID, "default"
			case id.Equal(i.nodeIdentity):
				return defaultID, "default"
			case i.localMembership.IsMe(id):
				info := i.localMembership.GetIdentityInfoByIdentity(BccspMSP, id)
				if info != nil {
					return id, info.ID
				}
				logger.Debugf("failed getting identity info for [%s:%s], returning the identity", BccspMSP, id)
				return id, ""
			case string(id) == "default":
				return defaultID, "default"
			}

			label := string(id)
			if idInfo := i.localMembership.GetIdentityInfoByLabel(BccspMSP, label); idInfo != nil {
				id, _, err := idInfo.GetIdentity()
				if err != nil {
					panic(fmt.Sprintf("failed getting identity [%s:%s] [%s]", BccspMSP, label, err))
				}
				return id, label
			}
			if idInfo := i.localMembership.GetIdentityInfoByIdentity(BccspMSP, id); idInfo != nil {
				return id, idInfo.ID
			}
			logger.Debugf("cannot match view.Identity string [%s] to identifier", vv)

			return id, ""
		case string:
			label := vv
			logger.Debugf("[x509] looking up identifier for label [%d,%s]", i.mspType, vv)
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
				info := i.localMembership.GetIdentityInfoByIdentity(BccspMSP, id)
				if info != nil {
					return id, info.ID
				}
				logger.Debugf("failed getting identity info for [%s:%s], returning the identity", BccspMSP, id)
				return id, ""
			}

			if idInfo := i.localMembership.GetIdentityInfoByLabel(BccspMSP, label); idInfo != nil {
				id, _, err := idInfo.GetIdentity()
				if err != nil {
					panic(fmt.Sprintf("failed getting identity [%s:%s] [%s]", BccspMSP, label, err))
				}
				return id, label
			}
			logger.Debugf("cannot match string [%s] to identifier", vv)
			return nil, label
		default:
			panic(fmt.Sprintf("identifier not recognised, expected []byte or view.Identity"))
		}
	case IdemixMSPIdentity:
		switch vv := v.(type) {
		case view.Identity:
			logger.Debugf("[idemix] looking up identifier for identity [%d,%s]", i.mspType, vv.String())
			id := vv
			switch {
			case id.IsNone():
				logger.Debugf("passed empty identity")
				return nil, "idemix"
			case id.Equal(defaultID):
				logger.Debugf("passed default identity")
				return nil, "idemix"
			case string(id) == "idemix":
				logger.Debugf("passed 'idemix' identity")
				return nil, "idemix"
			case id.Equal(i.nodeIdentity):
				logger.Debugf("passed identity is the node identity (same bytes)")
				return nil, "idemix"
			case i.localMembership.IsMe(id):
				logger.Debugf("passed identity is me")
				return id, ""
			}
			label := string(id)
			logger.Debugf("[idemix] looking up identifier for identity as label [%d,%s]", i.mspType, label)

			if idInfo := i.localMembership.GetIdentityInfoByLabel(IdemixMSP, label); idInfo != nil {
				return nil, idInfo.ID
			}
			// if idInfo := i.localMembership.GetIdentityInfoByIdentity(IdemixMSP, id); idInfo != nil {
			// 	return id, idInfo.ID
			// }
			logger.Debugf("cannot match view.Identity string [%s] to identifier", vv)
			return id, string(id)
		case string:
			label := vv
			logger.Debugf("[idemix] looking up identifier for label [%d,%s]", i.mspType, vv)
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

			if idInfo := i.localMembership.GetIdentityInfoByLabel(IdemixMSP, label); idInfo != nil {
				return nil, idInfo.ID
			}
			logger.Debugf("cannot match string [%s] to identifier", vv)
			return nil, label
		default:
			panic(fmt.Sprintf("identifier not recognised, expected []byte or view.Identity"))
		}
	default:
		panic(fmt.Sprintf("msp type [%d] not supported", i.mspType))
	}
}
