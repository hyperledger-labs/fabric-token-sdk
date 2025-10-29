/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"crypto/sha256"
	"reflect"

	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
)

var logger = logging.MustGetLogger()

type DeployerService interface {
	DeployTMSs() error
	DeployTMS(tmsID token.TMSID) error
	DeployTMSWithPP(tmsID token.TMSID, ppRaw []byte) error
}

func NewTMSDeployerService(
	ppFetcher *pp.PublicParametersService,
	configService *config.Service,
	nsSubmitter Submitter,
) *deployerService {
	return &deployerService{
		ppFetcher:     ppFetcher,
		configService: configService,
		nsSubmitter:   nsSubmitter,
		keyTranslator: &keys.Translator{},
	}
}

type deployerService struct {
	ppFetcher     fabric.NetworkPublicParamsFetcher
	configService *config.Service
	nsSubmitter   Submitter
	keyTranslator translator.KeyTranslator
}

func (s *deployerService) GetTMSIDs() ([]token.TMSID, error) {
	tmsIDs := make([]token.TMSID, 0)

	// TMSs
	tmss, err := s.configService.Configurations()
	if err != nil {
		logger.Errorf("Failed getting TMS configurations: %v", err)
		return nil, err
	}
	for _, tms := range tmss {
		tmsIDs = append(tmsIDs, tms.ID())
	}
	logger.Infof("Found %d namespaces under TMSs", len(tmss))

	return tmsIDs, nil
}

func (s *deployerService) DeployTMSs() error {
	logger.Infof("Deploying TMSs...")

	tmsIDs, err := s.GetTMSIDs()
	if err != nil {
		return err
	}

	logger.Infof("Found %d TMS IDs to deploy: [%v]", len(tmsIDs), tmsIDs)

	for _, tmsID := range tmsIDs {
		logger.Infof("Deploying TMS [%s]", tmsID)
		if err := s.DeployTMS(tmsID); err != nil {
			logger.Errorf("Failed deploying TMS [%s]: %v", tmsID, err)
			return err
		}
	}

	return nil
}

func (s *deployerService) DeployTMS(tmsID token.TMSID) error {
	return s.deployPublicParameters(tmsID)
}

func (s *deployerService) DeployTMSWithPP(tmsID token.TMSID, ppRaw []byte) error {
	return s.deployPublicParametersRaw(tmsID, ppRaw)
}

func (s *deployerService) deployPublicParameters(tmsID token.TMSID) error {
	ppRaw, err := s.ppFetcher.Fetch(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return err
	}
	return s.deployPublicParametersRaw(tmsID, ppRaw)
}

func (s *deployerService) deployPublicParametersRaw(tmsID token.TMSID, ppRaw []byte) error {
	tx, err := s.createPublicParametersTx(ppRaw, tmsID.Namespace)
	if err != nil {
		return err
	}
	return s.nsSubmitter.Submit(tmsID.Network, tmsID.Channel, tx)
}

func (s *deployerService) createPublicParametersTx(ppRaw []byte, namespaceID cdriver.Namespace) (*protoblocktx.Tx, error) {
	key, err := s.keyTranslator.CreateSetupKey()
	if err != nil {
		return nil, err
	}
	keyHash, err := s.keyTranslator.CreateSetupHashKey()
	if err != nil {
		return nil, err
	}

	valueHash := sha256.Sum256(ppRaw)
	tx := &protoblocktx.Tx{
		Namespaces: []*protoblocktx.TxNamespace{{
			NsId:        namespaceID,
			NsVersion:   0,
			ReadsOnly:   []*protoblocktx.Read{{Key: []byte("initialized")}},
			BlindWrites: []*protoblocktx.Write{{Key: []byte(key), Value: ppRaw}, {Key: []byte(keyHash), Value: valueHash[:]}},
		}},
	}

	return tx, nil
}

func GetTMSDeployerService(sp services.Provider) (DeployerService, error) {
	s, err := sp.GetService(reflect.TypeOf((*DeployerService)(nil)))
	if err != nil {
		return nil, err
	}
	return s.(DeployerService), nil
}
