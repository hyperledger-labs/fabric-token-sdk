/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type Manager[S any, D any, O dbDriver[D]] struct {
	logger  logging.Logger
	drivers map[drivers.DriverName]*dbOpener[S, D, O]
	cp      ConfigProvider
	config  Config

	mutex sync.Mutex
	dbs   map[string]S

	zero S
}

// DBByTMSId returns a service for the given TMS id
func (m *Manager[S, D, O]) DBByTMSId(id token.TMSID) (S, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Debugf("get service for [%s]", id)
	c, ok := m.dbs[id.String()]
	if ok {
		return c, nil
	}
	driverName, err := m.config.DriverFor(id)
	if err != nil {
		return m.zero, errors.Wrapf(err, "no driver found for [%s]", id)
	}
	d, ok := m.drivers[driverName]
	if !ok {
		return m.zero, errors.Errorf("no driver found for [%s]", driverName)
	}

	c, err = d.New(m.cp, id)
	if err != nil {
		return m.zero, errors.Wrapf(err, "failed instantiating service driver [%s]", driverName)
	}
	m.dbs[id.String()] = c

	return c, nil
}
