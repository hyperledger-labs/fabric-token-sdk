/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
)

var logger = logging.MustGetLogger("token-db")

type Manager[S any] struct{ lazy.Provider[token.TMSID, S] }

func newManager[V any](config *config.Service, prefix string, constructor func(name driver2.PersistenceName, params ...string) (V, error)) *Manager[V] {
	return &Manager[V]{Provider: lazy.NewProviderWithKeyMapper(Key, func(tmsID token.TMSID) (V, error) {
		logger.Infof("Creating manager for %T for [%s] and prefix [%s]", new(V), tmsID, prefix)
		cfg, err := config.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
		if err != nil {
			return utils.Zero[V](), err
		}

		return constructor(common.GetPersistenceName(cfg, prefix), tmsID.Network, tmsID.Channel, tmsID.Namespace)
	})}
}

func (m *Manager[S]) DBByTMSId(id token.TMSID) (S, error) {
	return m.Get(id)
}

func Key(tmsID token.TMSID) string {
	return tmsID.String()
}

func MappedManager[S any, T any](m *Manager[S], mapper func(S) (T, error)) *Manager[T] {
	return &Manager[T]{
		Provider: lazy.NewProviderWithKeyMapper(Key, func(tmsID token.TMSID) (T, error) {
			s, err := m.Get(tmsID)
			if err != nil {
				return utils.Zero[T](), err
			}
			return mapper(s)
		}),
	}
}
