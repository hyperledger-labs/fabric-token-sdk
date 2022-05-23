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

// QueryExecutor defines the interface for the query executor
type QueryExecutor struct {
	*auditdb.QueryExecutor
}

// Payments returns a filter for payments
func (a *QueryExecutor) Payments() *auditdb.PaymentsFilter {
	return a.QueryExecutor.NewPaymentsFilter()
}

// Holdings returns a filter for holdings
func (a *QueryExecutor) Holdings() *auditdb.HoldingsFilter {
	return a.QueryExecutor.NewHoldingsFilter()
}

// Done closes the query executor. It must be called when the query executor is no longer needed.
func (a *QueryExecutor) Done() {
	a.QueryExecutor.Done()
}

// Auditor is the interface for the auditor service
type Auditor struct {
	db *auditdb.AuditDB
}

// New returns a new Auditor instance for the passed auditor wallet
func New(sp view2.ServiceProvider, w *token.AuditorWallet) *Auditor {
	return &Auditor{db: auditdb.GetAuditDB(sp, w)}
}

// Validate validates the passed token request
func (a *Auditor) Validate(request *token.Request) error {
	return request.AuditCheck()
}

// Audit evaluates the passed token request and returns the list on inputs and outputs in the request
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

// NewQueryExecutor returns a new query executor
func (a *Auditor) NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{QueryExecutor: a.db.NewQueryExecutor()}
}
