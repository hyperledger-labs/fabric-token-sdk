/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger()

type StoreServiceManager[S any] interface {
	StoreServiceByTMSId(token.TMSID) (S, error)
}

type ServiceManager[S any] interface {
	ServiceByTMSId(token.TMSID) (S, error)
}

type manager[S any] struct{ lazy.Provider[token.TMSID, S] }

func NewStoreServiceManager[S any, T any](config *config.Service, prefix string, constructor func(name driver2.PersistenceName, params ...string) (S, error), mapper func(S) (T, error)) StoreServiceManager[T] {
	return &manager[T]{
		Provider: lazy.NewProviderWithKeyMapper(Key, func(tmsID token.TMSID) (T, error) {
			logger.Infof("Creating manager for %T for [%s] and prefix [%s]", new(T), tmsID, prefix)
			cfg, err := config.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
			if err != nil {
				return utils.Zero[T](), err
			}

			s, err := constructor(common.GetPersistenceName(cfg, prefix), tmsID.Network, tmsID.Channel, tmsID.Namespace)
			if err != nil {
				return utils.Zero[T](), err
			}
			return mapper(s)
		}),
	}
}

func (m *manager[S]) StoreServiceByTMSId(id token.TMSID) (S, error) { return m.Get(id) }

func Key(tmsID token.TMSID) string { return tmsID.String() }

func GetStoreService[S any, M StoreServiceManager[S]](sp token.ServiceProvider, tmsID token.TMSID) (S, error) {
	s, err := sp.GetService(new(M))
	if err != nil {
		return utils.Zero[S](), errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(M).StoreServiceByTMSId(tmsID)
	if err != nil {
		return utils.Zero[S](), errors.Wrapf(err, "failed to get store service for tms [%s]", tmsID)
	}
	return c, nil
}

func GetService[S any, M ServiceManager[S]](sp token.ServiceProvider, tmsID token.TMSID) (S, error) {
	s, err := sp.GetService(new(M))
	if err != nil {
		return utils.Zero[S](), errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(M).ServiceByTMSId(tmsID)
	if err != nil {
		return utils.Zero[S](), errors.Wrapf(err, "failed to get store service for tms [%s]", tmsID)
	}
	return c, nil
}
