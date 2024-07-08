/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	metrics2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
	"go.opentelemetry.io/otel/trace"
)

const currency = "CHF"

const (
	successLabel tracing.LabelName = "success"
)

var operationTypeMap = map[string]metrics.OperationType{
	"transfer":   metrics.Transfer,
	"balance":    metrics.Balance,
	"withdrawal": metrics.Withdraw,
}

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

func NewViewUser(username model.Username, auditor model.Username, client api2.ViewClient, idResolver idResolver, metrics *metrics.Metrics, tracerProvider trace.TracerProvider, logger logging.ILogger) *viewUser {
	return &viewUser{
		username:   username,
		auditor:    auditor,
		client:     client,
		idResolver: idResolver,
		metrics:    metrics,
		logger:     logger,
		tracer: tracerProvider.Tracer("user", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "token_sdk",
			LabelNames: []metrics2.MetricLabel{successLabel},
		})),
	}
}

type viewUser struct {
	username   model.Username
	auditor    model.Username
	client     api2.ViewClient
	idResolver idResolver
	metrics    *metrics.Metrics
	logger     logging.ILogger
	tracer     trace.Tracer
}

func (u *viewUser) CallView(fid string, in []byte) (interface{}, error) {
	return u.client.CallView(fid, in)
}

func (u *viewUser) Username() model.Username { return u.username }

func (u *viewUser) InitiateTransfer(_ api.Amount, _ api.UUID) api.Error { return nil }

func (u *viewUser) Transfer(value api.Amount, recipient model.Username, _ api.UUID) api.Error {
	u.logger.Infof("Call view for transfer of %d to %s\n", value, recipient)
	_, err := u.callView("transfer", &views.Transfer{
		Auditor:      u.auditor,
		Type:         currency,
		Amount:       uint64(value),
		Recipient:    u.idResolver.Identity(recipient),
		RecipientEID: recipient,
	})
	if err != nil {
		return api.NewInternalServerError(err, err.Error())
	}
	return nil
}

func (u *viewUser) Withdraw(value api.Amount) api.Error {
	u.logger.Infof("Call view to withdraw %d\n", value)

	_, err := u.callView("withdrawal", &views.Withdrawal{
		Wallet:    u.username,
		TokenType: currency,
		Amount:    uint64(value),
		Issuer:    "issuer",
	})
	if err != nil {
		return api.NewInternalServerError(err, err.Error())
	}
	return nil
}

func (u *viewUser) GetBalance() (api.Amount, api.Error) {
	u.logger.Infof("Call view to get balance of %s\n", u.username)

	res, err := u.callView("balance", &views.BalanceQuery{
		Wallet: u.username,
		Type:   currency,
	})
	if err != nil {
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

func (u *viewUser) callView(fid string, input interface{}) (interface{}, error) {
	marshaled, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	operationType := operationTypeMap[fid]

	ctx, span := u.tracer.Start(context.Background(), operationType)
	defer span.End()

	u.metrics.RequestsSent.
		With(metrics.OperationLabel, operationType).
		Add(1)

	start := time.Now()
	result, err := u.client.CallViewWithContext(ctx, fid, marshaled)

	span.SetAttributes(tracing.Bool(successLabel, err == nil))
	successType := metrics.SuccessValues[err == nil]
	u.metrics.RequestsReceived.
		With(metrics.OperationLabel, operationType, metrics.SuccessLabel, successType).
		Add(1)
	u.metrics.RequestDuration.
		With(metrics.OperationLabel, operationType, metrics.SuccessLabel, successType).
		Observe(time.Since(start).Seconds())

	if err != nil {
		u.logger.Warnf("Failed to call view %s: %v", fid, err)
	} else {
		u.logger.Infof("View %s completed successfully in %s\n", fid, time.Since(start))
	}

	return result, err
}
