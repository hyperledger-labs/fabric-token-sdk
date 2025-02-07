/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
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

	ppMutex sync.RWMutex
	ppMap   map[string]driver.PublicParameters
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
		ppMap:       map[string]driver.PublicParameters{},
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

func (s *SessionManager) PublicParameters(tds *core.TokenDriverService, namespace string) (driver.PublicParameters, error) {
	s.ppMutex.RLock()
	pp, ok := s.ppMap[namespace]
	s.ppMutex.RUnlock()
	if ok {
		return pp, nil
	}

	s.ppMutex.Lock()
	defer s.ppMutex.Unlock()

	pp, ok = s.ppMap[namespace]
	if ok {
		return pp, nil
	}

	ppRaw, err := s.ReadPublicParameters(namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read public parameters")
	}

	pp, err = tds.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	s.ppMap[namespace] = pp
	return pp, nil
}

func (s *SessionManager) ReadPublicParameters(namespace string) ([]byte, error) {
	for i := 0; i < 3; i++ {
		pp, err := s.readPublicParameters(namespace)
		if err != nil {
			logger.Errorf("failed to read public parameters from orion network [%s], retry [%d]", s.Orion.Name(), i)
			time.Sleep(100 * time.Minute)
			continue
		}
		return pp, nil
	}
	return nil, errors.Errorf("failed to read public parameters after 3 retries")
}

func (s *SessionManager) readPublicParameters(namespace string) ([]byte, error) {
	oSession, err := s.Orion.SessionManager().NewSession(s.CustodianID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session to orion network [%s]", s.Orion.Name())
	}
	qe, err := oSession.QueryExecutor(namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query executor for orion network [%s]", s.Orion.Name())
	}
	w := translator.New("", translator.NewRWSetWrapper(&ReadOnlyRWSWrapper{qe: qe}, "", ""), &translator.HashedKeyTranslator{KT: &keys.Translator{}})
	ppRaw, err := w.ReadSetupParameters()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve public parameters")
	}
	if len(ppRaw) == 0 {
		return nil, errors.Errorf("public parameters are not initiliazed yet")
	}
	logger.Debugf("public parameters read: %d", len(ppRaw))
	return ppRaw, nil
}
