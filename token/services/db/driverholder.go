/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"reflect"

	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/pkg/errors"
)

type ConfigProvider = driver.ConfigProvider

type Config interface {
	DriverFor(tmsID token.TMSID) (drivers.DriverName, error)
}

type serviceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

type dbDriver[D any] interface {
	Open(cp ConfigProvider, tmsid token.TMSID) (D, error)
}

type dbInstantiator[S any, D any, O dbDriver[D]] func(D) S

type dbOpener[S any, D any, O dbDriver[D]] struct {
	driver O
	newDB  dbInstantiator[S, D, O]
}

func (o *dbOpener[S, D, O]) New(cp ConfigProvider, id token.TMSID) (S, error) {
	driverInstance, err := o.driver.Open(cp, id)
	if err != nil {
		return utils.Zero[S](), err
	}
	return o.newDB(driverInstance), nil
}

type NamedDriver[O any] driver3.NamedDriver[O]

func NewDriverHolder[S any, D any, O dbDriver[D]](newDB dbInstantiator[S, D, O], ds ...NamedDriver[O]) *DriverHolder[S, D, O] {
	h := &DriverHolder[S, D, O]{
		Holder:      drivers.NewHolder[*dbOpener[S, D, O]](),
		managerType: reflect.TypeOf((*Manager[S, D, O])(nil)),
		zero:        utils.Zero[S](),
		newDB:       newDB,
	}
	for _, d := range ds {
		h.Register(string(d.Name), d.Driver)
	}
	return h
}

type DriverHolder[S any, D any, O dbDriver[D]] struct {
	*drivers.Holder[*dbOpener[S, D, O]]

	managerType reflect.Type
	zero        S

	newDB dbInstantiator[S, D, O]
}

func (h *DriverHolder[S, D, O]) Register(name drivers.DriverName, driver O) {
	h.Holder.Register(name, &dbOpener[S, D, O]{driver: driver, newDB: h.newDB})
}

// NewManager creates a new DB manager.
func (h *DriverHolder[S, D, O]) NewManager(cp ConfigProvider, config Config) *Manager[S, D, O] {
	return &Manager[S, D, O]{
		logger:  h.Logger,
		drivers: h.Drivers,
		cp:      cp,
		config:  config,
		dbs:     map[string]S{},

		zero: h.zero,
	}
}

// GetByTMSId returns the service for the given TMS id.
// Nil might be returned if the wallet is not found or an error occurred.
func (h *DriverHolder[S, D, O]) GetByTMSId(sp serviceProvider, tmsID token.TMSID) (S, error) {
	s, err := h.GetProvider(sp)
	if err != nil {
		return h.zero, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.DBByTMSId(tmsID)
	if err != nil {
		return h.zero, errors.Wrapf(err, "failed to get db for tms [%s]", tmsID)
	}
	return c, nil
}

func (h *DriverHolder[S, D, O]) GetProvider(sp serviceProvider) (*Manager[S, D, O], error) {
	s, err := sp.GetService(h.managerType)
	if err != nil {
		return utils.Zero[*Manager[S, D, O]](), errors.Wrapf(err, "failed to get manager service")
	}
	return s.(*Manager[S, D, O]), nil
}
