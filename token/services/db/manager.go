/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"crypto/sha256"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	logging2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	db2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-db")

type Manager[S any] struct{ lazy.Provider[token.TMSID, S] }

func NewManager[S any](cp driver2.ConfigProvider, drivers map[driver.PersistenceType]sql.Opener[S], configKeys ...string) *Manager[S] {
	return &Manager[S]{
		Provider: lazy.NewProviderWithKeyMapper(key, func(tmsID token.TMSID) (S, error) {
			opts, persistenceType, err := compileOpts(cp, tmsID, configKeys...)
			if err != nil {
				return utils.Zero[S](), errors.Wrapf(err, "failed to compile opts")
			}

			newDB, ok := drivers[persistenceType]
			if !ok {
				return utils.Zero[S](), errors.Errorf("no driver found for [%s], available: %s", persistenceType, logging2.Keys(drivers))
			}

			return newDB(opts)
		}),
	}
}

func compileOpts(cp driver2.ConfigProvider, tmsID token.TMSID, keys ...string) (common2.Opts, driver.PersistenceType, error) {
	tmsConfig, err := config2.NewService(cp).ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return common2.Opts{}, "", errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}

	for _, k := range keys {
		if !tmsConfig.IsSet(fmt.Sprintf("%s.type", k)) {
			logger.Infof("Key [%s] not found", k)
		} else if persistenceType := driver.PersistenceType(tmsConfig.GetString(fmt.Sprintf("%s.type", k))); persistenceType == mem.MemoryPersistence {
			return MemoryOpts(tmsID), mem.MemoryPersistence, nil
		} else if opts, err := sqlOpts(tmsConfig, k); err != nil {
			return common2.Opts{}, "", err
		} else {
			return opts, persistenceType, nil
		}
	}
	logger.Warnf("Persistence not found for keys [%v]. Defaulting to memory", keys)
	return MemoryOpts(tmsID), mem.MemoryPersistence, nil
}

func sqlOpts(tmsConfig config2.Configuration, k string) (common2.Opts, error) {
	opts, err := common2.GetOpts(tmsConfig, fmt.Sprintf("%s.opts", k))
	if err != nil {
		return common2.Opts{}, errors.Wrapf(err, "failed reading opts")
	}
	tmsID := tmsConfig.ID()
	opts.TablePrefix = db2.EscapeForTableName(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	return *opts, nil
}

func MemoryOpts(tmsID token.TMSID) common2.Opts {
	h := sha256.New()
	if _, err := h.Write([]byte(tmsID.String())); err != nil {
		panic(err)
	}
	o := mem.Opts
	o.DataSource = fmt.Sprintf("file:%x?mode=memory&cache=shared", h.Sum(nil))
	return o

}

func (m *Manager[S]) DBByTMSId(id token.TMSID) (S, error) {
	return m.Get(id)
}

func key(tmsID token.TMSID) string {
	return tmsID.String()
}

func MappedManager[S any, T any](m *Manager[S], mapper func(S) (T, error)) *Manager[T] {
	return &Manager[T]{
		Provider: lazy.NewProviderWithKeyMapper(key, func(tmsID token.TMSID) (T, error) {
			s, err := m.Get(tmsID)
			if err != nil {
				return utils.Zero[T](), err
			}
			return mapper(s)
		}),
	}
}
