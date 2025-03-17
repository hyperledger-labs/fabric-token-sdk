/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"

type KVS interface {
	Exists(id string) bool
	GetExisting(ids ...string) []string
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}
