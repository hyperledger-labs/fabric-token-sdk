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

// PublicParametersService models a service for fetching and loading public parameters.
type PublicParametersService struct {
	tmsProvider *token.ManagementServiceProvider
}

// NewPublicParametersService returns a new PublicParametersService instance.
func NewPublicParametersService(tmsProvider *token.ManagementServiceProvider) *PublicParametersService {
	return &PublicParametersService{tmsProvider: tmsProvider}
}

// LoadPublicParams loads the public parameters for the given TMS ID.
func (f *PublicParametersService) LoadPublicParams(tmsID token.TMSID, ppRaw []byte) error {
	return f.tmsProvider.Update(tmsID, ppRaw)
}

// Fetch returns the public parameters for the given network, channel, and namespace.
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

// Loader models a loader for public parameters.
type Loader interface {
	// LoadPublicParams loads the public parameters for the given TMS ID.
	LoadPublicParams(TMSID token.TMSID, ppRaw []byte) error
}

func GetPublicParametersService(sp services.Provider) (Loader, error) {
	s, err := sp.GetService(reflect.TypeOf((*Loader)(nil)))
	if err != nil {
		return nil, err
	}

	return s.(Loader), nil
}
