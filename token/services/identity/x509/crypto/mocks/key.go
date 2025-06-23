/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import "github.com/hyperledger/fabric-lib-go/bccsp"

type Key struct{}

func (t *Key) Bytes() ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (t *Key) SKI() []byte {
	// TODO implement me
	panic("implement me")
}

func (t *Key) Symmetric() bool {
	// TODO implement me
	panic("implement me")
}

func (t *Key) Private() bool {
	// TODO implement me
	panic("implement me")
}

func (t *Key) PublicKey() (bccsp.Key, error) {
	// TODO implement me
	panic("implement me")
}
