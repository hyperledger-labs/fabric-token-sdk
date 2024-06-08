/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package observables

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type transferMetrics struct {
	transferTracer trace.Tracer
}

func NewTransfer(p trace.TracerProvider) *transferMetrics {
	return &transferMetrics{
		transferTracer: p.Tracer("transfer", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:    "token_sdk",
			LabelNames:   metrics.AllLabelNames(SuccessfulLabel),
			StatsdFormat: metrics.StatsdFormat(SuccessfulLabel),
		})),
	}
}

type ObservableTransferService struct {
	TransferService driver.TransferService
	Metrics         *transferMetrics
}

func NewObservableTransferService(transferService driver.TransferService, metrics *transferMetrics) *ObservableTransferService {
	return &ObservableTransferService{TransferService: transferService, Metrics: metrics}
}

func (o *ObservableTransferService) Transfer(txID string, wallet driver.OwnerWallet, ids []*token.ID, Outputs []*token.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	_, span := o.Metrics.transferTracer.Start(context.Background(), "transfer")
	defer span.End()

	action, meta, err := o.TransferService.Transfer(txID, wallet, ids, Outputs, opts)
	span.SetAttributes(attribute.Bool(SuccessfulLabel, err == nil))
	return action, meta, err
}

func (o *ObservableTransferService) VerifyTransfer(tr driver.TransferAction, tokenInfos [][]byte) error {
	return o.TransferService.VerifyTransfer(tr, tokenInfos)
}

func (o *ObservableTransferService) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	return o.TransferService.DeserializeTransferAction(raw)
}
