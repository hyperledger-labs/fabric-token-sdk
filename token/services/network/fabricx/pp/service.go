/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"fmt"
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

type PublicParametersService struct {
	tmsProvider *token.ManagementServiceProvider
}

func NewPublicParametersService(tmsProvider *token.ManagementServiceProvider) *PublicParametersService {
	return &PublicParametersService{tmsProvider: tmsProvider}
}

func (f *PublicParametersService) LoadPublicParams(tmsID token.TMSID, ppRaw []byte) error {
	return f.tmsProvider.Update(tmsID, ppRaw)
}

func (f *PublicParametersService) Fetch(network driver.Network, channel driver.Channel, namespace driver.Namespace) ([]byte, error) {
	tmsID := token.TMSID{
		Network:   network,
		Channel:   channel,
		Namespace: namespace,
	}
	logger.Infof("Fetch public params for [%s]", tmsID)
	tms, err := f.tmsProvider.GetManagementService(token.WithTMSID(tmsID))
	if err != nil {
		return nil, fmt.Errorf("tms [%s] not found: %w", tmsID, err)
	}

	return tms.PublicParametersManager().PublicParameters().Serialize()
}

type Loader interface {
	LoadPublicParams(TMSID token.TMSID, ppRaw []byte) error
}

func GetPublicParametersService(sp services.Provider) (Loader, error) {
	s, err := sp.GetService(reflect.TypeOf((*Loader)(nil)))
	if err != nil {
		return nil, err
	}
	return s.(Loader), nil
}
