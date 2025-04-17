/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

var logger = logging.MustGetLogger("token-db")

type Manager[S any] struct{ lazy.Provider[token.TMSID, S] }

func newManager[V any](config *config.Service, prefix string, constructor func(cfg driver.Config, params ...string) (V, error)) *Manager[V] {
	return &Manager[V]{Provider: lazy.NewProviderWithKeyMapper(Key, func(tmsID token.TMSID) (V, error) {
		cfg, err := config.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
		if err != nil {
			return utils.Zero[V](), err
		}

		prefixConfig := db.NewPrefixConfig(cfg, prefix)
		if !prefixConfig.IsSet("") {
			logger.Warnf("Prefix [%s:%s] not found: changing to unity", tmsID, prefix)
			prefixConfig = db.NewPrefixConfig(cfg, "db.persistence")
		}
		if !prefixConfig.IsSet("") {
			logger.Errorf("unity not found for [%s] either", prefix)
			panic("no db driver found")
		}
		return constructor(prefixConfig, tmsID.Network, tmsID.Channel, tmsID.Namespace)
	})}
}

//func compileOpts(cp driver.ConfigProvider, tmsID token.TMSID, keys ...string) (common2.Opts, driver.PersistenceType, error) {
//	tmsConfig, err := config.NewService(cp).ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
//	if err != nil {
//		return common2.Opts{}, "", errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
//	}
//
//	for _, k := range keys {
//		if !tmsConfig.IsSet(fmt.Sprintf("%s.type", k)) {
//			logger.Infof("Key [%s] not found", k)
//		} else if persistenceType := driver.PersistenceType(tmsConfig.GetString(fmt.Sprintf("%s.type", k))); persistenceType == mem.MemoryPersistence {
//			return MemoryOpts(tmsID), mem.MemoryPersistence, nil
//		} else if opts, err := sqlOpts(tmsConfig, k); err != nil {
//			return common2.Opts{}, "", err
//		} else {
//			return opts, persistenceType, nil
//		}
//	}
//	logger.Warnf("Persistence not found for keys [%v]. Defaulting to memory", keys)
//	return MemoryOpts(tmsID), mem.MemoryPersistence, nil
//}
//
//func sqlOpts(tmsConfig config.Configuration, k string) (common2.Opts, error) {
//	opts, err := common2.GetOpts(tmsConfig, fmt.Sprintf("%s.opts", k))
//	if err != nil {
//		return common2.Opts{}, errors.Wrapf(err, "failed reading opts")
//	}
//	tmsID := tmsConfig.ID()
//	opts.TablePrefix = db2.EscapeForTableName(tmsID.Network, tmsID.Channel, tmsID.Namespace)
//	return *opts, nil
//}
//
//func MemoryOpts(tmsID token.TMSID) common2.Opts {
//	h := sha256.New()
//	if _, err := h.Write([]byte(tmsID.String())); err != nil {
//		panic(err)
//	}
//	o := mem.Opts
//	o.DataSource = fmt.Sprintf("file:%x?mode=memory&cache=shared", h.Sum(nil))
//	return o
//
//}

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
