/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
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
	IssueService
	TransferService
	TokenService
	AuditorService
	WalletService
	CertificationService
	Deserializer
	Serializer

	IdentityProvider() IdentityProvider
	Validator() (Validator, error)
	PublicParamsManager() PublicParamsManager
	ConfigManager() config.Manager
}

type TokenManagerServiceProvider interface {
	// GetTokenManagerService returns a TokenManagerService instance for the passed parameters
	// If a TokenManagerService is not available, it creates one.
	GetTokenManagerService(network string, channel string, namespace string, publicParamsFetcher PublicParamsFetcher) (TokenManagerService, error)
}
