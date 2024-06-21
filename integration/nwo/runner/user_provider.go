/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/constants"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
)

const currency = "CHF"

func newUserProvider(nw *integration.Infrastructure, metricsCollector metrics.Collector) (user.Provider, error) {
	fscTopology, err := getFscTopology(nw.Topologies)
	if err != nil {
		return nil, err
	}
	selector := make(token2.ReplicaSelector, len(fscTopology.Nodes))
	for _, node := range fscTopology.Nodes {
		selector[node.ID()] = node.Options.ReplicationFactor()
	}
	users := make(map[model.UserAlias][]*nwoUser, len(fscTopology.Nodes))
	for username, replicationFactor := range selector {
		replicas := make([]*nwoUser, replicationFactor)
		for i, replicaName := range selector.Get(username).AllNames() {
			client := nw.Client(replicaName)
			if client == nil {
				return nil, errors2.Errorf("could not find client for %s, only following found: [%v]", replicaName, collections.Keys(nw.Ctx.ViewClients))
			}
			replicas[i] = &nwoUser{
				username:         username,
				client:           client,
				idResolver:       nw,
				metricsCollector: metricsCollector,
			}
		}
		users[username] = replicas
	}
	return &nwoUserProvider{users: users}, nil
}

func getFscTopology(topologies []api2.Topology) (*fsc.Topology, error) {
	for _, t := range topologies {
		if fscTopology, ok := t.(*fsc.Topology); ok {
			return fscTopology, nil
		}
	}
	return nil, errors.New("fsc topology not found")
}

type nwoUserProvider struct {
	users map[model.UserAlias][]*nwoUser
}

func (p *nwoUserProvider) Get(alias model.UserAlias) user.User {
	if users, ok := p.users[alias]; ok {
		return users[0] // TODO Round robin
	}
	panic("did not find '" + alias + "', only following are available: " + strings.Join(collections.Keys(p.users), ", "))
}

type idResolver interface {
	Identity(model.Username) view.Identity
}

type nwoUser struct {
	username         model.Username
	client           api2.GRPCClient
	idResolver       idResolver
	metricsCollector metrics.Collector
}

func (u *nwoUser) Username() model.Username { return u.username }

func (u *nwoUser) InitiateTransfer(_ api.Amount, _ uuid.UUID) api.Error { return nil }

func (u *nwoUser) Transfer(value api.Amount, recipient model.Username, _ uuid.UUID) api.Error {
	u.metricsCollector.IncrementRequests()
	defer u.metricsCollector.DecrementRequests()
	start := time.Now()
	input, err := json.Marshal(&views.Transfer{
		Auditor:      "auditor",
		Type:         currency,
		Amount:       uint64(value),
		Recipient:    u.idResolver.Identity(recipient),
		RecipientEID: recipient,
	})
	if err != nil {
		return api.NewInternalServerError(err, err.Error())
	}
	_, err = u.client.CallView("transfer", input)
	u.metricsCollector.AddDuration(time.Since(start), constants.PaymentTransferRequest, err == nil)
	if err != nil {
		return api.NewInternalServerError(err, err.Error())
	}
	return nil
}

func (u *nwoUser) Withdraw(value api.Amount) api.Error {
	u.metricsCollector.IncrementRequests()
	defer u.metricsCollector.DecrementRequests()
	start := time.Now()
	input, err := json.Marshal(&views.Withdrawal{
		Wallet:    u.username,
		TokenType: currency,
		Amount:    uint64(value),
		Issuer:    "issuer",
	})
	if err != nil {
		return api.NewInternalServerError(err, err.Error())
	}
	_, err = u.client.CallView("withdrawal", input)
	u.metricsCollector.AddDuration(time.Since(start), constants.WithdrawRequest, err == nil)
	if err != nil {
		return api.NewInternalServerError(err, err.Error())
	}
	return nil
}

func (u *nwoUser) GetBalance() (api.Amount, api.Error) {
	u.metricsCollector.IncrementRequests()
	defer u.metricsCollector.DecrementRequests()
	input, err := json.Marshal(&views.BalanceQuery{
		Wallet: u.username,
		Type:   currency,
	})
	if err != nil {
		return 0, api.NewInternalServerError(err, err.Error())
	}
	res, err := u.client.CallView("balance", input)
	if err != nil {
		return 0, api.NewInternalServerError(err, err.Error())
	}

	b := &views.Balance{}
	if err := json.Unmarshal(res.([]byte), b); err != nil {
		return 0, api.NewInternalServerError(err, err.Error())
	}
	q, err := token.ToQuantity(b.Quantity, 64)
	if err != nil {
		return 0, api.NewInternalServerError(err, err.Error())
	}
	return q.ToBigInt().Int64(), nil
}
