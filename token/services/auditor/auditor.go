/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	db "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger("token-sdk.auditor")

// TxStatus is the status of a transaction
type TxStatus = auditdb.TxStatus

const (
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = auditdb.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = auditdb.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = auditdb.Deleted
)

const txIdLabel tracing.LabelName = "tx_id"

var TxStatusMessage = auditdb.TxStatusMessage

// Transaction models a generic token transaction
type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Namespace() string
	Request() *token.Request
}

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type CheckService interface {
	Check(context context.Context) ([]string, error)
}

// Auditor is the interface for the auditor service
type Auditor struct {
	networkProvider NetworkProvider
	tmsID           token.TMSID
	auditDB         *auditdb.DB
	tokenDB         *tokens.Tokens
	tmsProvider     TokenManagementServiceProvider
	finalityTracer  trace.Tracer
	checkService    CheckService
}

// Validate validates the passed token request
func (a *Auditor) Validate(context context.Context, request *token.Request) error {
	return request.AuditCheck(context)
}

// Audit extracts the list of inputs and outputs from the passed transaction.
// In addition, the Audit locks the enrollment named ids.
// Release must be invoked in case
func (a *Auditor) Audit(tx Transaction) (*token.InputStream, *token.OutputStream, error) {
	logger.Debugf("audit transaction [%s]....", tx.ID())
	request := tx.Request()
	record, err := request.AuditRecord()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting transaction audit record")
	}

	var eids []string
	eids = append(eids, record.Inputs.EnrollmentIDs()...)
	eids = append(eids, record.Outputs.EnrollmentIDs()...)
	logger.Debugf("audit transaction [%s], acquire locks", tx.ID())
	if err := a.auditDB.AcquireLocks(request.Anchor, eids...); err != nil {
		return nil, nil, err
	}
	logger.Debugf("audit transaction [%s], acquire locks done", tx.ID())

	return record.Inputs, record.Outputs, nil
}

// Append adds the passed transaction to the auditor database.
// It also releases the locks acquired by Audit.
func (a *Auditor) Append(tx Transaction) error {
	defer a.Release(tx)

	// append request to audit db
	if err := a.auditDB.Append(tx.Request(), a.gapFiller); err != nil {
		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// lister to events
	net, err := a.networkProvider.GetNetwork(tx.Network(), tx.Channel())
	if err != nil {
		return errors.WithMessagef(err, "failed getting network instance for [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("register tx status listener for tx [%s] at network [%s]", tx.ID(), tx.Network())
	var r driver.FinalityListener = common.NewFinalityListener(logger, a.tmsProvider, a.tmsID, a.auditDB, a.tokenDB, a.finalityTracer)
	if err := net.AddFinalityListener(tx.Namespace(), tx.ID(), r); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("append done for request [%s]", tx.ID())
	return nil
}

// Release releases the lock acquired of the passed transaction.
func (a *Auditor) Release(tx Transaction) {
	a.auditDB.ReleaseLocks(tx.Request().Anchor)
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *Auditor) SetStatus(ctx context.Context, txID string, status db.TxStatus, message string) error {
	newCtx, span := a.finalityTracer.Start(ctx, "set_status", tracing.WithAttributes(tracing.String(txIdLabel, txID)))
	defer span.End()
	return a.auditDB.SetStatus(newCtx, txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *Auditor) GetStatus(txID string) (TxStatus, string, error) {
	return a.auditDB.GetStatus(txID)
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (a *Auditor) GetTokenRequest(txID string) ([]byte, error) {
	return a.auditDB.GetTokenRequest(txID)
}

func (a *Auditor) Check(context context.Context) ([]string, error) {
	return a.checkService.Check(context)
}

func (a *Auditor) gapFiller(record *token.AuditRecord) error {
	filter := record.Inputs.ByEnrollmentID("")
	if filter.Count() == 0 {
		return nil
	}
	// TODO: extract from the audit tokens
	targetEID := record.Outputs.EnrollmentIDs()[0]

	// fetch all the tokens
	tms, err := a.tmsProvider.GetManagementService(token.WithTMSID(a.tmsID))
	if err != nil {
		return errors.WithMessagef(err, "failed getting management service for [%s]", a.tmsID)
	}
	tokens, err := tms.Vault().NewQueryEngine().ListAuditTokens(filter.IDs()...)
	if err != nil {
		return errors.WithMessagef(err, "failed listing tokens for [%s]", filter.IDs())
	}
	precision := tms.PublicParametersManager().PublicParameters().Precision()
	for i := 0; i < filter.Count(); i++ {
		item := filter.At(i)
		item.EnrollmentID = targetEID
		item.Owner = tokens[i].Owner
		item.Type = tokens[i].Type
		q, err := token2.ToQuantity(tokens[i].Quantity, precision)
		if err != nil {
			return errors.WithMessagef(err, "failed converting token quantity [%s]", tokens[i].Quantity)
		}
		item.Quantity = q
	}
	return nil
}
