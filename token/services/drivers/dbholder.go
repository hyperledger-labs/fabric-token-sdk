/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package drivers

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type ConfigProvider = driver.ConfigProvider

type ServiceProvider interface {
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

func NewDBHolder[S any, D any, O dbDriver[D]](newDB dbInstantiator[S, D, O]) *DBDriverHolder[S, D, O] {
	return &DBDriverHolder[S, D, O]{
		ServiceHolder: newServiceHolder[S, *dbOpener[S, D, O]](),
		newDB:         newDB,
	}
}

type DBDriverHolder[S any, D any, O dbDriver[D]] struct {
	*ServiceHolder[S, *dbOpener[S, D, O]]

	newDB dbInstantiator[S, D, O]
}

func (h *DBDriverHolder[S, D, O]) Register(name DriverName, driver O) {
	h.Holder.Register(name, &dbOpener[S, D, O]{driver: driver, newDB: h.newDB})
}

func (h *DBDriverHolder[S, D, O]) GetByTMSId(sp ServiceProvider, tmsID token.TMSID) (S, error) {
	return h.ServiceHolder.GetByTMSId(sp, tmsID)
}

func (h *DBDriverHolder[S, D, O]) NewManager(cp ConfigProvider, config Config) *DBManager[S, D, O] {
	return &DBManager[S, D, O]{ServiceManager: h.ServiceHolder.NewManager(cp, config)}
}

type DBManager[S any, D any, O dbDriver[D]] struct {
	*ServiceManager[S, *dbOpener[S, D, O]]
}

func (m *DBManager[S, D, O]) DBByTMSId(id token.TMSID) (S, error) {
	return m.ServiceManager.ServiceByTMSId(id)
}
