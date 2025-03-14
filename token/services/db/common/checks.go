/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenTransactionDB interface {
	GetTokenRequest(txID string) ([]byte, error)
	Transactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error)
}

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type Checker = func(context context.Context) ([]string, error)

type NamedChecker struct {
	Name    string
	Checker Checker
}

type ChecksService struct {
	checkers []NamedChecker
}

func NewChecksService(checkers []NamedChecker) *ChecksService {
	return &ChecksService{checkers: checkers}
}

func (a *ChecksService) Check(context context.Context) ([]string, error) {
	var errorMessages []string
	for _, checker := range a.checkers {
		errs, err := checker.Checker(context)
		if err != nil {
			return nil, errors.Wrapf(err, "failed checking with checker [%s]", checker.Name)
		}
		errorMessages = append(errorMessages, errs...)
	}
	return errorMessages, nil
}

type DefaultCheckers struct {
	tmsProvider     TokenManagementServiceProvider
	networkProvider NetworkProvider
	db              TokenTransactionDB
	tokenDB         *tokens.Tokens
	tmsID           token.TMSID
}

func NewDefaultCheckers(tmsProvider TokenManagementServiceProvider, networkProvider NetworkProvider, db TokenTransactionDB, tokenDB *tokens.Tokens, tmsID token.TMSID) []NamedChecker {
	checkers := &DefaultCheckers{tmsProvider: tmsProvider, networkProvider: networkProvider, db: db, tokenDB: tokenDB, tmsID: tmsID}
	return []NamedChecker{
		{
			Name:    "Transaction Check",
			Checker: checkers.checkTransactions,
		},
		{
			Name:    "Unspent Tokens Check",
			Checker: checkers.checkUnspentTokens,
		},
		{
			Name:    "Token Spendability Check",
			Checker: checkers.checkTokenSpendability,
		},
	}
}

func (a *DefaultCheckers) checkTransactions(context context.Context) ([]string, error) {
	var errorMessages []string

	tms, err := a.tmsProvider.GetManagementService(token.WithTMSID(a.tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting tms [%s]", a.tmsID)
	}
	net, err := a.networkProvider.GetNetwork(tms.Network(), tms.Channel())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network [%s]", tms.ID())
	}
	l, err := net.Ledger()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ledger [%s]", tms.ID())
	}

	it, err := a.db.Transactions(driver.QueryTransactionsParams{})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed querying transactions [%s]", tms.ID())
	}
	defer it.Close()
	for {
		transactionRecord, err := it.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed querying transactions [%s]", tms.ID())
		}
		if transactionRecord == nil {
			break
		}

		tokenRequest, err := a.db.GetTokenRequest(transactionRecord.TxID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting token request [%s]", transactionRecord.TxID)
		}
		if tokenRequest == nil {
			return nil, errors.Errorf("token request [%s] is nil", transactionRecord.TxID)
		}

		// check the ledger
		lVC, _, err := l.Status(transactionRecord.TxID)
		if err != nil {
			lVC = network.Unknown
		}
		switch {
		case transactionRecord.Status == driver.Confirmed && lVC != network.Valid:
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
			}
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is valid for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		case transactionRecord.Status == driver.Deleted && lVC != network.Invalid:
			if lVC != network.Unknown || transactionRecord.Status != driver.Deleted {
				if err != nil {
					errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
				}
				errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is invalid for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
			}
		case transactionRecord.Status == driver.Unknown && lVC != network.Unknown:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is unknown for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		case transactionRecord.Status == driver.Pending && lVC == network.Busy:
			// this is fine, let's continue
		case transactionRecord.Status == driver.Pending && lVC != network.Unknown:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is busy for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		}
	}
	return errorMessages, nil
}

func (a *DefaultCheckers) checkUnspentTokens(context context.Context) ([]string, error) {
	var errorMessages []string

	tms, err := a.tmsProvider.GetManagementService(token.WithTMSID(a.tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting tms [%s]", a.tmsID)
	}
	net, err := a.networkProvider.GetNetwork(tms.Network(), tms.Channel())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network [%s]", tms.ID())
	}
	tv, err := net.TokenVault(tms.Namespace())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault [%s]", tms.ID())
	}
	uit, err := tv.QueryEngine().UnspentTokensIterator()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed querying utxo engine")
	}
	defer uit.Close()
	var unspentTokenIDs []*token2.ID
	for {
		tok, err := uit.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed querying next unspent token")
		}
		if tok == nil {
			break
		}
		unspentTokenIDs = append(unspentTokenIDs, tok.Id)
	}
	ledgerTokenContent, err := net.QueryTokens(context, tms.Namespace(), unspentTokenIDs)
	if err != nil {
		errorMessages = append(errorMessages, fmt.Sprintf("failed to query tokens: [%s]", err))
	} else {
		if len(unspentTokenIDs) != len(ledgerTokenContent) {
			return nil, errors.Errorf("length diffrence")
		}
		index := 0
		if err := tv.QueryEngine().GetTokenOutputs(unspentTokenIDs, func(id *token2.ID, tokenRaw []byte) error {
			for _, content := range ledgerTokenContent {
				if bytes.Equal(content, tokenRaw) {
					return nil
				}
			}

			errorMessages = append(errorMessages, fmt.Sprintf("token content does not match at [%s][%d], [%s]", id, index, hash.Hashable(tokenRaw)))
			index++
			return nil
		}); err != nil {
			return nil, errors.WithMessagef(err, "failed to match ledger token content with local")
		}
	}
	return errorMessages, nil
}

func (a *DefaultCheckers) checkTokenSpendability(context context.Context) ([]string, error) {
	var errorMessages []string

	tms, err := a.tmsProvider.GetManagementService(token.WithTMSID(a.tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting tms [%s]", a.tmsID)
	}
	net, err := a.networkProvider.GetNetwork(tms.Network(), tms.Channel())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network [%s]", tms.ID())
	}
	tv, err := net.TokenVault(tms.Namespace())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault [%s]", tms.ID())
	}
	uit, err := tv.QueryEngine().UnspentTokensIterator()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed querying utxo engine")
	}
	defer uit.Close()

	// supportedtokenFormats := tms.TokensService().SupportedTokenFormats()

	for {
		tok, err := uit.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed querying next unspent token")
		}
		if tok == nil {
			break
		}
		// is the token's format supported?

		// extract the token's recipients and try to get un unmarshaller

	}

	return errorMessages, nil
}
