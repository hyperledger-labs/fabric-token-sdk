/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"fmt"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/pkg/errors"
)

type DBManager struct {
	OrionNetworkProvider *orion.NetworkServiceProvider
	IsCustodian          bool
	ConfigProvider       configProvider

	SMMutex         sync.RWMutex
	SessionManagers map[string]*SessionManager
}

func NewDBManager(onp *orion.NetworkServiceProvider, cp configProvider, isCustodian bool) *DBManager {
	return &DBManager{
		OrionNetworkProvider: onp,
		ConfigProvider:       cp,
		IsCustodian:          isCustodian,
		SessionManagers:      map[string]*SessionManager{},
	}
}

func (d *DBManager) GetSessionManager(network string) (*SessionManager, error) {
	d.SMMutex.RLock()
	sm, ok := d.SessionManagers[network]
	d.SMMutex.RUnlock()
	if ok {
		return sm, nil
	}

	d.SMMutex.Lock()
	defer d.SMMutex.Unlock()

	sm, ok = d.SessionManagers[network]
	if ok {
		return sm, nil
	}
	sm, err := NewSessionManager(d, network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating session manager for network [%s]", network)
	}
	d.SessionManagers[network] = sm
	return sm, nil
}

type SessionManager struct {
	dbManager   *DBManager
	Orion       *orion.NetworkService
	CustodianID string

	reuse         bool
	reuseOnce     sync.Once
	sharedSession *orion.Session
}

func NewSessionManager(dbManager *DBManager, network string) (*SessionManager, error) {
	custodianID, err := GetCustodian(dbManager.ConfigProvider, network)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	ons, err := dbManager.OrionNetworkProvider.NetworkService(network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting ons network service for [%s]", network)
	}
	return &SessionManager{
		dbManager:   dbManager,
		Orion:       ons,
		CustodianID: custodianID,
		reuse:       true,
	}, nil
}

func (s *SessionManager) GetSession() (os *orion.Session, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failed getting session for [%s]: [%s]", s.CustodianID, r)
		}
	}()
	if s.reuse {
		logger.Debugf("reuse session to orion [%s]", s.CustodianID)
		s.reuseOnce.Do(func() {
			oSession, err := s.Orion.SessionManager().NewSession(s.CustodianID)
			if err != nil {
				panic(fmt.Sprintf("failed to create session to orion network [%s]: [%s]", s.CustodianID, err))
			}
			s.sharedSession = oSession
		})
		return s.sharedSession, nil
	}

	logger.Debugf("open session to orion [%s]", s.CustodianID)
	oSession, err := s.Orion.SessionManager().NewSession(s.CustodianID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session to orion network [%s]", s.CustodianID)
	}
	return oSession, nil
}
