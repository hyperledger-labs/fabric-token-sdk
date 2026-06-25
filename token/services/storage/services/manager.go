/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package services

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

type ConfigService interface {
	ConfigurationFor(network string, channel string, namespace string) (driver.Configuration, error)
}

type ServiceManager[S any] interface {
	ServiceByTMSId(token.TMSID) (S, error)
}

type manager[T any] struct{ lazy.Provider[token.TMSID, T] }

func NewServiceManager[T any](constructor func(tmsID token.TMSID) (T, error)) ServiceManager[T] {
	return &manager[T]{
		Provider: lazy.NewProviderWithKeyMapper(services.Key, func(tmsID token.TMSID) (T, error) {
			logger.Infof("Creating manager for %T for [%s]", new(T), tmsID)
			s, err := constructor(tmsID)
			if err != nil {
				return utils.Zero[T](), err
			}

			return s, nil
		}),
	}
}

func (m *manager[T]) ServiceByTMSId(id token.TMSID) (T, error) { return m.Get(id) }
