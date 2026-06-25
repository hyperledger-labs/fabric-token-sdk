/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package libp2p

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	common "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/host"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/host/libp2p"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/provider"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
)

type SDK struct {
	common.SDK
}

func NewFrom(sdk common.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	if err := p.Container().Decorate(CustomHostProvider); err != nil {
		return err
	}

	// Call the parent
	if err := p.SDK.Install(); err != nil {
		return err
	}

	return nil
}

// CustomHostProvider extends the default host provider with libp2p support.
func CustomHostProvider(config driver.ConfigService, endpointService *endpoint.Service, metricsProvider metrics.Provider, tracerProvider tracing.Provider) (host.GeneratorProvider, error) {
	p2pCommType := strings.ToLower(config.GetString("fsc.p2p.type"))
	switch p2pCommType {
	case libp2p.P2PCommunicationType:
		if err := endpointService.AddPublicKeyExtractor(&comm.PKExtractor{}); err != nil {
			return nil, err
		}
		endpointService.SetPublicKeyIDSynthesizer(&libp2p.PKIDSynthesizer{})

		return libp2p.NewHostGeneratorProvider(libp2p.NewConfig(config), metricsProvider, endpointService), nil
	default:
		return provider.NewHostProvider(config, endpointService, metricsProvider, tracerProvider)
	}
}
