/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package auditor

import (
	"github.com/pkg/errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb"
)

type QueryExecutor struct {
	*auditdb.QueryExecutor
}

func (a *QueryExecutor) Payments() *auditdb.PaymentsFilter {
	return a.QueryExecutor.NewPaymentsFilter()
}

func (a *QueryExecutor) Holdings() *auditdb.HoldingsFilter {
	return a.QueryExecutor.NewHoldingsFilter()
}

func (a *QueryExecutor) Done() {
	a.QueryExecutor.Done()
}

type Auditor struct {
	db *auditdb.AuditDB
}

func New(sp view2.ServiceProvider, w *token.AuditorWallet) *Auditor {
	return &Auditor{db: auditdb.GetAuditDB(sp, w)}
}

func (a *Auditor) Validate(request *token.Request) error {
	return request.AuditCheck()
}

func (a *Auditor) Audit(request *token.Request) (*token.InputStream, *token.OutputStream, error) {
	inputs, err := request.AuditInputs()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting inputs")
	}
	outputs, err := request.AuditOutputs()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting outputs")
	}

	return inputs, outputs, nil
}

func (a *Auditor) NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{QueryExecutor: a.db.NewQueryExecutor()}
}
