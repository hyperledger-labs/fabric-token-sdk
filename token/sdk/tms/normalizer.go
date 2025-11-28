/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

type ConfigService interface {
	Configurations() ([]driver.Configuration, error)
}

type tmsNormalizer struct {
	configService ConfigService
	normalizer    token.Normalizer
}

func NewTMSNormalizer(tmsProvider ConfigService, normalizer token.Normalizer) *tmsNormalizer {
	return &tmsNormalizer{
		configService: tmsProvider,
		normalizer:    normalizer,
	}
}

func (p *tmsNormalizer) Normalize(opt *token.ServiceOptions) (*token.ServiceOptions, error) {
	// lookup configurations
	configs, err := p.configService.Configurations()
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting tms configs")
	}
	if len(configs) == 0 {
		return nil, errors.Errorf("no token management service configs found")
	}

	logger.Debugf("normalizing opts [%v]", opt)
	if len(opt.Network) != 0 {
		// filter configurations by network
		configs = Filter(configs, func(c driver.Configuration) bool {
			return c.ID().Network == opt.Network
		})
		if len(configs) == 0 {
			return nil, errors.Errorf("no token management service config found for network [%s]", opt.Network)
		}
	}

	if len(opt.Channel) != 0 {
		// filter configurations by channel
		configs = Filter(configs, func(c driver.Configuration) bool {
			return c.ID().Channel == opt.Channel
		})
		if len(configs) == 0 {
			return nil, errors.Errorf("no token management service config found for network and channel [%s:%s]", opt.Network, opt.Channel)
		}
	}

	if len(opt.Namespace) != 0 {
		// filter configurations by namespace
		configs = Filter(configs, func(c driver.Configuration) bool {
			return c.ID().Namespace == opt.Namespace
		})
		if len(configs) == 0 {
			return nil, errors.Errorf("no token management service config found for network, channel, and namespace [%s:%s:%s]", opt.Network, opt.Channel, opt.Namespace)
		}
	}

	// if we reach this point there must be at least one configuration
	logger.Debugf("found [%d] matching configurations for opts [%v], take the first one", len(configs), opt)
	id := configs[0].ID()
	opt.Network = id.Network
	opt.Channel = id.Channel
	opt.Namespace = id.Namespace

	// last pass
	return p.normalizer.Normalize(opt)
}

// Filter keeps elements where keep(x) == true, allocating a new result slice.
func Filter[E any](in []E, keep func(E) bool) []E {
	out := make([]E, 0, len(in))
	for _, x := range in {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}
