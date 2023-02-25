/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

// TMSID models a TMS identifier
type TMSID struct {
	Network   string
	Channel   string
	Namespace string
}

// String returns a string representation of the TMSID
func (t TMSID) String() string {
	return fmt.Sprintf("%s,%s,%s", t.Network, t.Channel, t.Namespace)
}

// ServiceProvider is used to return instances of a given type
type ServiceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

// ServiceOptions is used to configure the service
type ServiceOptions struct {
	// Network is the name of the network
	Network string
	// Channel is the name of the channel, if meaningful for the underlying backend
	Channel string
	// Namespace is the namespace of the token
	Namespace string
	// PublicParamsFetcher is used to fetch the public parameters
	PublicParamsFetcher PublicParamsFetcher
	// Params is used to store any application specific parameter
	Params map[string]interface{}
}

// TMSID returns the TMSID for the given ServiceOptions
func (o ServiceOptions) TMSID() TMSID {
	return TMSID{
		Network:   o.Network,
		Channel:   o.Channel,
		Namespace: o.Namespace,
	}
}

// ParamAsString returns the value bound to the passed key.
// If the key is not found, it returns the empty string.
// if the value bound to the passed key is not a string, it returns an error.
func (o ServiceOptions) ParamAsString(key string) (string, error) {
	if o.Params == nil {
		return "", nil
	}
	v, ok := o.Params[key]
	if !ok {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.Errorf("expecting string, found [%T]", o)
	}
	return s, nil
}

// CompileServiceOptions compiles the given list of ServiceOption
func CompileServiceOptions(opts ...ServiceOption) (*ServiceOptions, error) {
	txOptions := &ServiceOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

// ServiceOption is a function that configures a ServiceOptions
type ServiceOption func(*ServiceOptions) error

// WithNetwork sets the network name
func WithNetwork(network string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Network = network
		return nil
	}
}

// WithChannel sets the channel
func WithChannel(channel string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Channel = channel
		return nil
	}
}

// WithNamespace sets the namespace for the service
func WithNamespace(namespace string) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Namespace = namespace
		return nil
	}
}

// WithPublicParameterFetcher sets the public parameter fetcher
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

// ManagementService (TMS, for short) is the entry point for the Token API. A TMS is uniquely
// identified by a network, channel, namespace, and public parameters.
// The TMS gives access, among other things, to the wallet manager, the public paramenters,
// the token selector, and so on.
type ManagementService struct {
	sp        view.ServiceProvider
	network   string
	channel   string
	namespace string
	tms       driver.TokenManagerService

	vaultProvider               VaultProvider
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	signatureService            *SignatureService
	vault                       *Vault
}

// String returns a string representation of the TMS
func (t *ManagementService) String() string {
	return fmt.Sprintf("TMS[%s:%s:%s]", t.Network(), t.Channel(), t.Network())
}

// Network returns the network identifier
func (t *ManagementService) Network() string {
	return t.network
}

// Channel returns the channel identifier
func (t *ManagementService) Channel() string {
	return t.channel
}

// Namespace returns the namespace identifier, empty if not defined
func (t *ManagementService) Namespace() string {
	return t.namespace
}

// NewRequest returns a new Token Request whose anchor is the passed id
func (t *ManagementService) NewRequest(id string) (*Request, error) {
	return NewRequest(t, id), nil
}

// NewRequestFromBytes returns a new Token Request for the passed anchor, and whose actions and metadata are
// unmarshalled from the passed bytes
func (t *ManagementService) NewRequestFromBytes(anchor string, actions []byte, meta []byte) (*Request, error) {
	return NewRequestFromBytes(t, anchor, actions, meta)
}

// NewMetadataFromBytes unmarshals the passed bytes into a Metadata object
func (t *ManagementService) NewMetadataFromBytes(raw []byte) (*Metadata, error) {
	tokenRequestMetadata := &driver.TokenRequestMetadata{}
	if err := tokenRequestMetadata.FromBytes(raw); err != nil {
		return nil, err
	}
	return &Metadata{
		TMS:                  t.tms,
		TokenRequestMetadata: tokenRequestMetadata,
	}, nil
}

// Validator returns a new token validator for this TMS
func (t *ManagementService) Validator() (*Validator, error) {
	v, err := t.tms.Validator()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get validator")
	}
	return &Validator{backend: v}, nil
}

// Vault returns the Token Vault for this TMS
func (t *ManagementService) Vault() *Vault {
	return t.vault
}

// WalletManager returns the wallet manager for this TMS
func (t *ManagementService) WalletManager() *WalletManager {
	return &WalletManager{managementService: t}
}

// CertificationManager returns the certification manager for this TMS
func (t *ManagementService) CertificationManager() *CertificationManager {
	return &CertificationManager{c: t.tms}
}

// CertificationClient returns the certification client for this TMS
func (t *ManagementService) CertificationClient() (*CertificationClient, error) {
	certificationClient, err := t.certificationClientProvider.New(
		t.Network(), t.Channel(), t.Namespace(), t.PublicParametersManager().CertificationDriver(),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create ceritifacation client")
	}
	return &CertificationClient{cc: certificationClient}, nil
}

// PublicParametersManager returns a manager that gives access to the public parameters
// governing this TMS.
func (t *ManagementService) PublicParametersManager() *PublicParametersManager {
	return &PublicParametersManager{ppm: t.tms.PublicParamsManager()}
}

// SelectorManager returns a manager that gives access to the token selectors
func (t *ManagementService) SelectorManager() SelectorManager {
	return t.selectorManagerProvider.SelectorManager(t.Network(), t.Channel(), t.Namespace())
}

// SigService returns the signature service for this TMS
func (t *ManagementService) SigService() *SignatureService {
	return t.signatureService
}

// ID returns the TMS identifier
func (t *ManagementService) ID() TMSID {
	return TMSID{
		Network:   t.Network(),
		Channel:   t.Channel(),
		Namespace: t.Namespace(),
	}
}

// ConfigManager returns the configuration manager for this TMS
func (t *ManagementService) ConfigManager() *ConfigManager {
	return &ConfigManager{cm: t.tms.ConfigManager()}
}

func (t *ManagementService) init() error {
	v, err := t.vaultProvider.Vault(t.network, t.channel, t.namespace)
	if err != nil {
		return errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", t.namespace, t.channel, t.namespace)
	}
	t.vault = &Vault{v: v}
	return nil
}

// GetManagementService returns the management service for the passed options. If no options are passed,
// the default management service is returned.
// Options: WithNetwork, WithChannel, WithNamespace, WithPublicParameterFetcher, WithTMS, WithTMSID
// The function panics if an error occurs. Use GetManagementServiceProvider(sp).GetManagementService(opts...) to handle any error directly
func GetManagementService(sp ServiceProvider, opts ...ServiceOption) *ManagementService {
	ms, err := GetManagementServiceProvider(sp).GetManagementService(opts...)
	if err != nil {
		logger.Warnf("failed to get token manager service [%s]", err)
		return nil
	}
	return ms
}

// NewServicesFromPublicParams uses the passed marshalled public parameters to create an instance
// of PublicParametersManager and a new instance of Validator.
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

	if err := ppm.Validate(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to validate public parameters")
	}

	logger.Debugf("instantiate validator...")
	validator, err := core.NewValidator(pp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed instantiating validator")
	}

	return &PublicParametersManager{ppm: ppm}, &Validator{backend: validator}, nil
}

func NewPublicParametersManagerFromPublicParams(params []byte) (*PublicParametersManager, error) {
	logger.Debugf("unmarshall public parameters...")
	pp, err := core.PublicParametersFromBytes(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling public parameters")
	}

	logger.Debugf("instantiate public parameters manager...")
	ppm, err := core.NewPublicParametersManager(pp)
	if err != nil {
		return nil, errors.Wrap(err, "failed instantiating public parameters manager")
	}

	return &PublicParametersManager{ppm: ppm}, nil
}
