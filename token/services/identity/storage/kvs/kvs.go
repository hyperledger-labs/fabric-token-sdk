/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
)

type KVS interface {
	Exists(ctx context.Context, id string) bool
	GetExisting(ctx context.Context, ids ...string) []string
	Put(ctx context.Context, id string, state interface{}) error
	Get(ctx context.Context, id string, state interface{}) error
	GetByPartialCompositeID(ctx context.Context, prefix string, attrs []string) (kvs.Iterator, error)
}
