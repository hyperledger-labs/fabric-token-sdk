/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package validator

//go:generate counterfeiter -o mock/ledger.go -fake-name Ledger . Ledger

type Ledger interface {
	GetState(key string) ([]byte, error)
}
