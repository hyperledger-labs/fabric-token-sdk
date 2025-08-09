/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
)

var managerType = reflect.TypeOf((*ServiceManager)(nil))

type StoreServiceManager db.StoreServiceManager[*tokendb.StoreService]

type TMSProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

// ServiceManager handles the services
type ServiceManager struct {
	p lazy.Provider[token.TMSID, *Service]
}

// NewServiceManager creates a new Service manager.
func NewServiceManager(
	tmsProvider TMSProvider,
	storeServiceManager StoreServiceManager,
	networkProvider NetworkProvider,
	notifier events.Publisher,
) *ServiceManager {
	return &ServiceManager{
		p: lazy.NewProviderWithKeyMapper(db.Key, func(tmsID token.TMSID) (*Service, error) {
			db, err := storeServiceManager.StoreServiceByTMSId(tmsID)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get tokendb for [%s]", tmsID)
			}

			storage, err := NewDBStorage(notifier, db, tmsID)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get token store for [%s]", tmsID)
			}
			tokens := &Service{
				TMSProvider:     tmsProvider,
				NetworkProvider: networkProvider,
				Storage:         storage,
				RequestsCache:   secondcache.NewTyped[*CacheEntry](5000),
			}
			return tokens, nil
		}),
	}
}

// ServiceByTMSId returns the Service for the given TMS
func (cm *ServiceManager) ServiceByTMSId(tmsID token.TMSID) (*Service, error) {
	return cm.p.Get(tmsID)
}

// GetService returns the Service instance for the passed TMS
func GetService(sp token.ServiceProvider, tmsID token.TMSID) (*Service, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get manager service")
	}
	if err != nil {
		return nil, err
	}
	tokens, err := s.(*ServiceManager).ServiceByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}
