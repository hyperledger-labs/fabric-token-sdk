/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

var storageProviderType = reflect.TypeOf((*StorageProvider)(nil))

//go:generate counterfeiter -o dep/mock/storage.go -fake-name Storage . Storage

type Storage interface {
	Append(ctx context.Context, tx *Transaction) error
}

//go:generate counterfeiter -o dep/mock/storage_provider.go -fake-name StorageProvider . StorageProvider

type StorageProvider interface {
	GetStorage(id token.TMSID) (Storage, error)
	CacheRequest(ctx context.Context, tmsID token.TMSID, request *token.Request) error
}

func GetStorageProvider(sp token.ServiceProvider) (StorageProvider, error) {
	s, err := sp.GetService(storageProviderType)
	if err != nil {
		return nil, err
	}
	return s.(StorageProvider), nil
}

// StoreTransactionRecords stores the transaction records extracted from the passed transaction to the
// token transaction db
func StoreTransactionRecords(ctx view.Context, tx *Transaction) error {
	sp, err := GetStorageProvider(ctx)
	if err != nil {
		return errors.Join(ErrStorage, err)
	}
	s, err := sp.GetStorage(tx.TMS.ID())
	if err != nil {
		return errors.Join(ErrStorage, err)
	}
	if err := s.Append(ctx.Context(), tx); err != nil {
		return errors.Join(ErrStorage, err)
	}
	return nil
}
