/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var (
	managementServiceProviderIndex = &ManagementServiceProvider{}
)

type Normalizer interface {
	Normalize(opt *ServiceOptions) *ServiceOptions
}

type VaultProvider interface {
	Vault(network string, channel string, namespace string) tokenapi.Vault
}

type SelectorManager interface {
	NewSelector(id string) (Selector, error)
	Unlock(txID string) error
}

type SelectorManagerProvider interface {
	SelectorManager(network string, channel string, namespace string) SelectorManager
}

type CertificationClientProvider interface {
	New(network string, channel string, namespace string, driver string) (tokenapi.CertificationClient, error)
}

type ManagementServiceProvider struct {
	sp                          ServiceProvider
	tmsProvider                 tokenapi.TokenManagerServiceProvider
	normalizer                  Normalizer
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	vaultProvider               VaultProvider
}

func NewManagementServiceProvider(
	sp ServiceProvider,
	tmsProvider tokenapi.TokenManagerServiceProvider,
	normalizer Normalizer,
	vaultProvider VaultProvider,
	certificationClientProvider CertificationClientProvider,
	selectorManagerProvider SelectorManagerProvider,
) *ManagementServiceProvider {
	return &ManagementServiceProvider{
		sp:                          sp,
		tmsProvider:                 tmsProvider,
		normalizer:                  normalizer,
		vaultProvider:               vaultProvider,
		certificationClientProvider: certificationClientProvider,
		selectorManagerProvider:     selectorManagerProvider,
	}
}

func (p *ManagementServiceProvider) GetManagementService(opts ...ServiceOption) *ManagementService {
	opt, err := CompileServiceOptions(opts...)
	if err != nil {
		panic(err)
	}
	opt = p.normalizer.Normalize(opt)

	logger.Debugf("get tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)
	tokenService, err := p.tmsProvider.GetTokenManagerService(
		opt.Network,
		opt.Channel,
		opt.Namespace,
		opt.PublicParamsFetcher,
	)
	if err != nil {
		logger.Errorf("failed getting TMS for [%s]: [%s]", opt.TMSID(), err)
		return nil
	}

	logger.Debugf("returning tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)

	return &ManagementService{
		sp:                          p.sp,
		network:                     opt.Network,
		channel:                     opt.Channel,
		namespace:                   opt.Namespace,
		tms:                         tokenService,
		vaultProvider:               p.vaultProvider,
		certificationClientProvider: p.certificationClientProvider,
		selectorManagerProvider:     p.selectorManagerProvider,
		signatureService: &SignatureService{
			deserializer: tokenService,
			ip:           tokenService.IdentityProvider(),
		},
	}
}

func GetManagementServiceProvider(sp ServiceProvider) *ManagementServiceProvider {
	s, err := sp.GetService(managementServiceProviderIndex)
	if err != nil {
		panic(err)
	}
	return s.(*ManagementServiceProvider)
}
