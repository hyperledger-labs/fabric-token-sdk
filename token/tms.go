/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// TMSID models a TMS identifier
type TMSID struct {
	Network   string
	Channel   string
	Namespace string
}

func (t *TMSID) String() string {
	return fmt.Sprintf("%s,%s,%s", t.Network, t.Channel, t.Namespace)
}

// ServiceProvider is used to return instances of a given type
type ServiceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

type Info struct {
	TokenDataHiding bool
	GraphHiding     bool
}

type ServiceOptions struct {
	Network             string
	Channel             string
	Namespace           string
	PublicParamsFetcher PublicParamsFetcher
}

func (o ServiceOptions) TMSID() TMSID {
	return TMSID{
		Network:   o.Network,
		Channel:   o.Channel,
		Namespace: o.Namespace,
	}
}

func CompileServiceOptions(opts ...ServiceOption) (*ServiceOptions, error) {
	txOptions := &ServiceOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

type ServiceOption func(*ServiceOptions) error

func WithNetwork(network string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Network = network
		return nil
	}
}

func WithChannel(channel string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Channel = channel
		return nil
	}
}

func WithNamespace(namespace string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Namespace = namespace
		return nil
	}
}

func WithPublicParameterFetcher(ppFetcher PublicParamsFetcher) ServiceOption {
	return func(o *ServiceOptions) error {
		o.PublicParamsFetcher = ppFetcher
		return nil
	}
}

// WithTMS filters by network, channel and namespace. Each of them can be empty
func WithTMS(network, channel, namespace string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Network = network
		o.Channel = channel
		o.Namespace = namespace
		return nil
	}
}

// WithTMSID filters by TMS identifier
func WithTMSID(id TMSID) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Network = id.Network
		o.Channel = id.Channel
		o.Namespace = id.Namespace
		return nil
	}
}

type ManagementService struct {
	sp        view.ServiceProvider
	network   string
	channel   string
	namespace string
	tms       tokenapi.TokenManagerService

	vaultProvider               VaultProvider
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	signatureService            *SignatureService
}

func (t *ManagementService) String() string {
	return fmt.Sprintf("TMS[%s:%s:%s]", t.Network(), t.Channel(), t.Network())
}

func (t *ManagementService) Network() string {
	return t.network
}

func (t *ManagementService) Channel() string {
	return t.channel
}

func (t *ManagementService) Namespace() string {
	return t.namespace
}

func (t *ManagementService) NewRequest(txId string) (*Request, error) {
	return NewRequest(t, txId), nil
}

func (t *ManagementService) NewRequestFromBytes(txId string, requestRaw []byte, metaRaw []byte) (*Request, error) {
	return NewRequestFromBytes(t, txId, requestRaw, metaRaw)
}

func (t *ManagementService) NewMetadataFromBytes(raw []byte) (*Metadata, error) {
	tokenRequestMetadata := &tokenapi.TokenRequestMetadata{}
	if err := tokenRequestMetadata.FromBytes(raw); err != nil {
		return nil, err
	}
	return &Metadata{
		queryService:         t.tms,
		tokenRequestMetadata: tokenRequestMetadata,
	}, nil
}

func (t *ManagementService) Validator() *Validator {
	return &Validator{backend: t.tms.Validator()}
}

func (t *ManagementService) Vault() *Vault {
	return &Vault{v: t.vaultProvider.Vault(t.network, t.channel, t.namespace)}
}

func (t *ManagementService) WalletManager() *WalletManager {
	return &WalletManager{ts: t.tms}
}

func (t *ManagementService) CertificationManager() *CertificationManager {
	return &CertificationManager{c: t.tms}
}

func (t *ManagementService) CertificationClient() *CertificationClient {
	certificationClient, err := t.certificationClientProvider.New(
		t.Network(), t.Channel(), t.Namespace(), t.PublicParametersManager().CertificationDriver(),
	)
	if err != nil {
		panic(err)
	}
	return &CertificationClient{cc: certificationClient}
}

func (t *ManagementService) PublicParametersManager() *PublicParametersManager {
	return &PublicParametersManager{ppm: t.tms.PublicParamsManager()}
}

func (t *ManagementService) SelectorManager() SelectorManager {
	return t.selectorManagerProvider.SelectorManager(t.Network(), t.Channel(), t.Namespace())
}

func (t *ManagementService) SigService() *SignatureService {
	return t.signatureService
}

func (t *ManagementService) ID() TMSID {
	return TMSID{
		Network:   t.Network(),
		Channel:   t.Channel(),
		Namespace: t.Namespace(),
	}
}

func (t *ManagementService) ConfigManager() *ConfigManager {
	return &ConfigManager{cm: t.tms.ConfigManager()}
}

func GetManagementService(sp ServiceProvider, opts ...ServiceOption) *ManagementService {
	return GetManagementServiceProvider(sp).GetManagementService(opts...)
}

func NewServicesFromPublicParams(params []byte) (*PublicParametersManager, *Validator, error) {
	logger.Debugf("unmarshall public parameters...")
	pp, err := core.PublicParametersFromBytes(params)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling public parameters")
	}

	logger.Debugf("instantiate public parameters manager...")
	ppm, err := core.NewPublicParametersManager(pp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed instantiating public parameters manager")
	}

	logger.Debugf("instantiate validator...")
	validator, err := core.NewValidator(pp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed instantiating validator")
	}

	return &PublicParametersManager{ppm: ppm}, &Validator{backend: validator}, nil
}
