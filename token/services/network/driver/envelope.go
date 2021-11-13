/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type Envelope interface {
	Results() []byte
	Bytes() ([]byte, error)
	FromBytes([]byte) error
	TxID() string
	Nonce() []byte
	Creator() []byte
}
