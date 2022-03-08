/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
)

type RWSWrapper struct {
	qe *orion.SessionQueryExecutor
}

func (r *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	panic("programming error: this should not be called")
}

func (r *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	res, _, err := r.qe.Get(key)
	return res, err
}

func (r *RWSWrapper) DeleteState(namespace string, key string) error {
	panic("programming error: this should not be called")
}

func (r *RWSWrapper) Bytes() ([]byte, error) {
	panic("programming error: this should not be called")
}

func (r *RWSWrapper) Done() {
	panic("programming error: this should not be called")
}

func (r *RWSWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	panic("programming error: this should not be called")
}

func (r *RWSWrapper) Equals(right interface{}, namespace string) error {
	panic("programming error: this should not be called")
}
