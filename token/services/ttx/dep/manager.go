/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dep

import (
	"context"

	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
)

// CheckService defines the interface for the check service.
//
//go:generate counterfeiter -o mock/check_service.go -fake-name CheckService . CheckService
type CheckService interface {
	// Check performs a consistency check on the underlying database.
	Check(ctx context.Context) ([]string, error)
}

// CheckServiceProvider defines the interface for the check service provider.
//
//go:generate counterfeiter -o mock/check_service_provider.go -fake-name CheckServiceProvider . CheckServiceProvider
type CheckServiceProvider interface {
	// CheckService returns a CheckService instance for the given TMS ID and related services.
	CheckService(id token.TMSID, adb StoreService, tdb TokensService) (CheckService, error)
}

// TokensService defines the interface for the tokens service.
// It manages the lifecycle and caching of token requests.
//
//go:generate counterfeiter -o mock/tokens_service.go -fake-name TokensService . TokensService
type TokensService interface {
	// CacheRequest stores the given request in the cache.
	CacheRequest(ctx context.Context, tmsID token.TMSID, request *token.Request) error
	// GetCachedTokenRequest returns the cached token request for the given transaction ID.
	GetCachedTokenRequest(txID string) (*token.Request, []byte)
	// Append appends the given request to the storage.
	Append(ctx context.Context, tmsID token.TMSID, txID token.RequestAnchor, request *token.Request) error
}

// TokensServiceManager defines the interface for the tokens service manager.
//
//go:generate counterfeiter -o mock/tokens_service_manager.go -fake-name TokensServiceManager . TokensServiceManager
type TokensServiceManager interface {
	// ServiceByTMSId returns the TokensService instance for the given TMS ID.
	ServiceByTMSId(token.TMSID) (TokensService, error)
}

// StoreService defines the interface for the ttx store service.
// It provides methods for querying and managing transaction records and token requests.
//
//go:generate counterfeiter -o mock/store_service.go -fake-name StoreService . StoreService
type StoreService interface {
	// Transactions returns a list of transaction records that match the given criteria.
	Transactions(ctx context.Context, params ttxdb.QueryTransactionsParams, pagination cdriver.Pagination) (*cdriver.PageIterator[*storage.TransactionRecord], error)
	// TokenRequests returns an iterator over the token requests matching the passed params.
	TokenRequests(ctx context.Context, params ttxdb.QueryTokenRequestsParams) (driver.TokenRequestIterator, error)
	// AppendTransactionRecord appends the transaction record for the given request.
	AppendTransactionRecord(ctx context.Context, req *token.Request) error
	// SetStatus sets the status of the transaction with the given ID.
	SetStatus(ctx context.Context, txID string, status storage.TxStatus, message string) error
	// GetStatus returns the status of the transaction with the given ID.
	GetStatus(ctx context.Context, txID string) (storage.TxStatus, string, error)
	// GetTokenRequest returns the token request for the given transaction ID.
	GetTokenRequest(ctx context.Context, txID string) ([]byte, error)
	// AddTransactionEndorsementAck records an endorsement acknowledgement for the given transaction ID.
	AddTransactionEndorsementAck(ctx context.Context, txID string, id token.Identity, sigma []byte) error
	// GetTransactionEndorsementAcks returns all endorsement acknowledgements for the given transaction ID.
	GetTransactionEndorsementAcks(ctx context.Context, txID string) (map[string][]byte, error)
}

// StoreServiceManager defines the interface for the ttx store service manager.
//
//go:generate counterfeiter -o mock/store_service_manager.go -fake-name StoreServiceManager . StoreServiceManager
type StoreServiceManager interface {
	// StoreServiceByTMSId returns the StoreService instance for the given TMS ID.
	StoreServiceByTMSId(token.TMSID) (StoreService, error)
}
