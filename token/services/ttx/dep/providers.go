/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dep

import (
	"context"
	"reflect"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

var (
	networkProviderType         = reflect.TypeOf((*NetworkProvider)(nil))
	tmsProviderType             = reflect.TypeOf((*TokenManagementServiceProvider)(nil))
	networkIdentityProviderType = reflect.TypeOf((*NetworkIdentityProvider)(nil))
)

//go:generate counterfeiter -o mock/network.go -fake-name Network . Network

type Network interface {
	AddFinalityListener(namespace string, txID string, listener network.FinalityListener) error
	NewEnvelope() *network.Envelope
	AnonymousIdentity() (view.Identity, error)
	LocalMembership() *network.LocalMembership
	ComputeTxID(n *network.TxID) string
}

//go:generate counterfeiter -o mock/network_provider.go -fake-name NetworkProvider . NetworkProvider

type NetworkProvider interface {
	GetNetwork(network string, channel string) (Network, error)
}

func GetNetworkProvider(sp token.ServiceProvider) (NetworkProvider, error) {
	s, err := sp.GetService(networkProviderType)
	if err != nil {
		return nil, err
	}
	return s.(NetworkProvider), nil
}

type SignatureService interface {
	IsMe(ctx context.Context, party token.Identity) bool
}

//go:generate counterfeiter -o mock/tms.go -fake-name TokenManagementService . TokenManagementService

type TokenManagementService interface {
	ID() token.TMSID
	Network() string
	Channel() string
	NewRequest(anchor token.RequestAnchor) (*token.Request, error)
	SelectorManager() (token.SelectorManager, error)
	PublicParametersManager() *token.PublicParametersManager
	SigService() *token.SignatureService
	WalletManager() *token.WalletManager
	NewFullRequestFromBytes(raw []byte) (*token.Request, error)
	Vault() *token.Vault
}

//go:generate counterfeiter -o mock/tmse.go -fake-name TokenManagementServiceWithExtensions . TokenManagementServiceWithExtensions

type TokenManagementServiceWithExtensions interface {
	TokenManagementService
	SetTokenManagementService(req *token.Request) error
}

//go:generate counterfeiter -o mock/tmsp.go -fake-name TokenManagementServiceProvider . TokenManagementServiceProvider

type TokenManagementServiceProvider interface {
	TokenManagementService(opts ...token.ServiceOption) (TokenManagementServiceWithExtensions, error)
}

func GetManagementService(sp token.ServiceProvider, opts ...token.ServiceOption) (TokenManagementServiceWithExtensions, error) {
	s, err := sp.GetService(tmsProviderType)
	if err != nil {
		return nil, err
	}
	tmsProvider, ok := s.(TokenManagementServiceProvider)
	if !ok {
		panic("implementation error, type must be TokenManagementServiceProvider")
	}
	return tmsProvider.TokenManagementService(opts...)
}

//go:generate counterfeiter -o mock/nis.go -fake-name NetworkIdentitySigner . NetworkIdentitySigner

type NetworkIdentitySigner = driver2.Signer

//go:generate counterfeiter -o mock/nip.go -fake-name NetworkIdentityProvider . NetworkIdentityProvider

type NetworkIdentityProvider interface {
	DefaultIdentity() view.Identity
	GetSigner(identity view.Identity) (NetworkIdentitySigner, error)
}

func GetNetworkIdentityProvider(sp token.ServiceProvider) (NetworkIdentityProvider, error) {
	s, err := sp.GetService(networkIdentityProviderType)
	if err != nil {
		return nil, err
	}
	nip, ok := s.(NetworkIdentityProvider)
	if !ok {
		panic("implementation error, type must be NetworkIdentityProvider")
	}
	return nip, nil
}
