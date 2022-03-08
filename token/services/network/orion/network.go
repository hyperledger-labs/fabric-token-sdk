/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Network struct {
	sp view2.ServiceProvider
	n  *orion.NetworkService
}

func NewNetwork(sp view2.ServiceProvider, n *orion.NetworkService) *Network {
	return &Network{sp: sp, n: n}
}

func (n Network) Name() string {
	return n.n.Name()
}

func (n Network) Channel() string {
	panic("channels not supported")
}

func (n Network) Vault(namespace string) (driver.Vault, error) {
	panic("implement me")
}

func (n Network) GetRWSet(id string, results []byte) (driver.RWSet, error) {
	panic("implement me")
}

func (n Network) StoreEnvelope(id string, env []byte) error {
	panic("implement me")
}

func (n Network) Broadcast(blob interface{}) error {
	panic("implement me")
}

func (n Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	panic("implement me")
}

func (n Network) IsFinal(id string) error {
	panic("implement me")
}

func (n Network) NewEnvelope() driver.Envelope {
	panic("implement me")
}

func (n Network) StoreTransient(id string, transient driver.TransientMap) error {
	panic("implement me")
}

func (n Network) RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	panic("implement me")
}

func (n Network) ComputeTxID(id *driver.TxID) string {
	panic("implement me")
}

func (n Network) FetchPublicParameters(namespace string) ([]byte, error) {
	pp, err := view2.GetManager(n.sp).InitiateView(NewPublicParamsRequestView(n.Name(), namespace))
	if err != nil {
		return nil, err
	}
	return pp.([]byte), nil
}

func (n Network) QueryTokens(context view.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
	panic("implement me")
}

func (n Network) LocalMembership() driver.LocalMembership {
	panic("implement me")
}

func (n Network) GetEnrollmentID(raw []byte) (string, error) {
	panic("implement me")
}
