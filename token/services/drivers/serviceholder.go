/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package drivers

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type DriverName = string

type serviceDriver[S any] interface {
	New(cp ConfigProvider, tmsID token.TMSID) (S, error)
}

type ServiceHolder[S any, D serviceDriver[S]] struct {
	*Holder[D]

	managerType reflect.Type
	zero        S
}

func newServiceHolder[S any, D serviceDriver[S]]() *ServiceHolder[S, D] {
	return &ServiceHolder[S, D]{
		Holder:      NewHolder[D](),
		managerType: reflect.TypeOf((*ServiceManager[S, D])(nil)),
		zero:        utils.Zero[S](),
	}
}

type Config interface {
	DriverFor(tmsID token.TMSID) (DriverName, error)
}

// ServiceManager handles the services
type ServiceManager[S any, D serviceDriver[S]] struct {
	logger  logging.Logger
	drivers map[DriverName]D
	cp      ConfigProvider
	config  Config

	mutex sync.Mutex
	dbs   map[string]S

	zero S
}

// NewManager creates a new service manager.
func (h *ServiceHolder[S, D]) NewManager(cp ConfigProvider, config Config) *ServiceManager[S, D] {
	return &ServiceManager[S, D]{
		logger:  h.logger,
		drivers: h.Drivers,
		cp:      cp,
		config:  config,
		dbs:     map[string]S{},

		zero: h.zero,
	}
}

// ServiceByTMSId returns a service for the given TMS id
func (m *ServiceManager[S, D]) ServiceByTMSId(id token.TMSID) (S, error) {
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

// GetByTMSId returns the service for the given TMS id.
// Nil might be returned if the wallet is not found or an error occurred.
func (h *ServiceHolder[S, D]) GetByTMSId(sp ServiceProvider, tmsID token.TMSID) (S, error) {
	s, err := h.GetProvider(sp)
	if err != nil {
		return h.zero, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.ServiceByTMSId(tmsID)
	if err != nil {
		return h.zero, errors.Wrapf(err, "failed to get db for tms [%s]", tmsID)
	}
	return c, nil
}

func (h *ServiceHolder[S, D]) GetProvider(sp ServiceProvider) (*ServiceManager[S, D], error) {
	s, err := sp.GetService(h.managerType)
	if err != nil {
		return utils.Zero[*ServiceManager[S, D]](), errors.Wrapf(err, "failed to get manager service")
	}
	return s.(*ServiceManager[S, D]), nil
}
