/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
)

type RWSetProcessor struct {
	nss       []string
	sp        view2.ServiceProvider
	ownership network.Ownership
	issued    network.Issued
}

func NewTokenRWSetProcessor(ns string, sp view2.ServiceProvider, ownership network.Ownership, issued network.Issued) *RWSetProcessor {
	return &RWSetProcessor{
		nss:       []string{ns},
		sp:        sp,
		ownership: ownership,
		issued:    issued,
	}
}

func (r *RWSetProcessor) Process(req orion.Request, tx orion.ProcessTransaction, rws *orion.RWSet, ns string) error {
	return nil
}
