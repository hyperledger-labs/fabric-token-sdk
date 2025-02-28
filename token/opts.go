/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

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
	// PublicParams contains the public params to use to instantiate the driver
	PublicParams []byte
	// Params is used to store any application specific parameter
	Params map[string]interface{}
	// Initiator is the view initiating the service
	Initiator view.View
	// Duration is the duration a given operation should take
	Duration time.Duration
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

// WithPublicParameterFetcher sets the public parameters fetcher
func WithPublicParameterFetcher(ppFetcher PublicParamsFetcher) ServiceOption {
	return func(o *ServiceOptions) error {
		o.PublicParamsFetcher = ppFetcher
		return nil
	}
}

// WithPublicParameter sets the public parameters
func WithPublicParameter(publicParams []byte) ServiceOption {
	return func(o *ServiceOptions) error {
		o.PublicParams = publicParams
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

// WithTMSIDPointer filters by TMS identifier, if provided
func WithTMSIDPointer(id *TMSID) ServiceOption {
	return func(o *ServiceOptions) error {
		if id == nil {
			return nil
		}
		o.Network = id.Network
		o.Channel = id.Channel
		o.Namespace = id.Namespace
		return nil
	}
}

// WithInitiator sets the view initiating the service
func WithInitiator(initiator view.View) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Initiator = initiator
		return nil
	}
}

// WithDuration sets the duration a given operation should take
func WithDuration(duration time.Duration) ServiceOption {
	return func(o *ServiceOptions) error {
		o.Duration = duration
		return nil
	}
}
