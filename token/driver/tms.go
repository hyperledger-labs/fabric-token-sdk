/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"fmt"

	"github.com/pkg/errors"
)

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

func (t TMSID) Equal(tmsid TMSID) bool {
	return t.Network == tmsid.Network && t.Channel == tmsid.Channel && t.Namespace == tmsid.Namespace
}

// TokenManagerService is the entry point of the Driver API and gives access to the rest of the API
type TokenManagerService interface {
	IssueService() IssueService
	TransferService() TransferService
	TokensService() TokensService
	AuditorService() AuditorService
	CertificationService() CertificationService
	Deserializer() Deserializer
	Serializer() Serializer
	IdentityProvider() IdentityProvider
	Validator() (Validator, error)
	PublicParamsManager() PublicParamsManager
	Configuration() Configuration
	WalletService() WalletService
	Authorization() Authorization
	// Done releases all the resources allocated by this service
	Done() error
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
	// PublicParams contains the public params to use to instantiate the driver
	PublicParams []byte
	// Params is used to store any application specific parameter
	Params map[string]interface{}
}

func (o ServiceOptions) String() string {
	return fmt.Sprintf("%s,%s,%s", o.Network, o.Channel, o.Namespace)
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

type TokenManagerServiceProvider interface {
	// GetTokenManagerService returns a TokenManagerService instance for the passed parameters
	// If a TokenManagerService is not available, it creates one.
	GetTokenManagerService(opts ServiceOptions) (TokenManagerService, error)

	// NewTokenManagerService returns a new TokenManagerService instance for the passed parameters
	NewTokenManagerService(opts ServiceOptions) (TokenManagerService, error)

	Update(options ServiceOptions) error

	Configurations() ([]Configuration, error)

	PublicParametersFromBytes(raw []byte) (PublicParameters, error)
}
