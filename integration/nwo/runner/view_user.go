/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"encoding/json"
	"strings"
	"time"

	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/constants"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
)

const currency = "CHF"

type ViewUserProvider struct {
	users map[model.UserAlias][]user.User
}

func NewViewUserProvider(users map[model.UserAlias][]user.User) *ViewUserProvider {
	return &ViewUserProvider{users: users}
}

func (p *ViewUserProvider) Get(alias model.UserAlias) user.User {
	if users, ok := p.users[alias]; ok {
		return users[0] // TODO Round robin
	}
	panic("did not find '" + alias + "', only following are available: " + strings.Join(collections.Keys(p.users), ", "))
}

type idResolver interface {
	Identity(model.Username) view.Identity
}

func NewViewUser(username model.Username, auditor model.Username, client api2.ViewClient, idResolver idResolver, metricsCollector metrics.Collector, logger logging.ILogger) *viewUser {
	return &viewUser{
		username:         username,
		auditor:          auditor,
		client:           client,
		idResolver:       idResolver,
		metricsCollector: metricsCollector,
		logger:           logger,
	}
}

type viewUser struct {
	username         model.Username
	auditor          model.Username
	client           api2.ViewClient
	idResolver       idResolver
	metricsCollector metrics.Collector
	logger           logging.ILogger
}

func (u *viewUser) CallView(fid string, in []byte) (interface{}, error) {
	return u.client.CallView(fid, in)
}

func (u *viewUser) Username() model.Username { return u.username }

func (u *viewUser) InitiateTransfer(_ api.Amount, _ api.UUID) api.Error { return nil }

func (u *viewUser) Transfer(value api.Amount, recipient model.Username, _ api.UUID) api.Error {
	u.logger.Infof("Call view for transfer of %d to %s\n", value, recipient)
	u.metricsCollector.IncrementRequests()
	defer u.metricsCollector.DecrementRequests()
	start := time.Now()
	input, err := json.Marshal(&views.Transfer{
		Auditor:      u.auditor,
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
		u.logger.Errorf("Failed to call view transfer: %s", err)
		return api.NewInternalServerError(err, err.Error())
	}
	u.logger.Infof("Transfer of %d to %s took %s", value, recipient, time.Since(start))
	return nil
}

func (u *viewUser) Withdraw(value api.Amount) api.Error {
	u.logger.Infof("Call view to withdraw %d\n", value)
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
		u.logger.Errorf("Failed to call view withdrawal: %s", err)
		return api.NewInternalServerError(err, err.Error())
	}
	u.logger.Infof("Successfully completed withdrawal")
	return nil
}

func (u *viewUser) GetBalance() (api.Amount, api.Error) {
	u.logger.Infof("Call view to get balance of %s\n", u.username)
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
		u.logger.Errorf("Failed to call view balance: %s", err)
		return 0, api.NewInternalServerError(err, err.Error())
	}

	b := &views.Balance{}
	if err := json.Unmarshal(res.([]byte), b); err != nil {
		return 0, api.NewInternalServerError(err, err.Error())
	}
	u.logger.Infof("Received balance result: [%s]", b.Quantity)
	q, err := token.ToQuantity(b.Quantity, 64)
	if err != nil {
		return 0, api.NewInternalServerError(err, err.Error())
	}
	return q.ToBigInt().Int64(), nil
}
