/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ObservableTransferService struct {
	TransferService driver.TransferService
	Metrics         *Metrics
}

func NewObservableTransferService(transferService driver.TransferService, metrics *Metrics) *ObservableTransferService {
	return &ObservableTransferService{TransferService: transferService, Metrics: metrics}
}

func (o *ObservableTransferService) Transfer(txID string, wallet driver.OwnerWallet, ids []*token.ID, Outputs []*token.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	action, meta, err := o.TransferService.Transfer(txID, wallet, ids, Outputs, opts)
	o.Metrics.AddTransfer(err == nil)
	return action, meta, err
}

func (o *ObservableTransferService) VerifyTransfer(tr driver.TransferAction, tokenInfos [][]byte) error {
	//TODO implement me
	panic("implement me")
}

func (o *ObservableTransferService) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	//TODO implement me
	panic("implement me")
}
