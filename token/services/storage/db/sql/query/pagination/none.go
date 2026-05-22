/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pagination

import "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"

type empty struct{}

// Empty returns a pagination instance that will force a query an empty page
func Empty() *empty {
	return &empty{}
}

func (p *empty) Prev() (driver.Pagination, error) {
	return p, nil
}

func (p *empty) Next() (driver.Pagination, error) {
	return &empty{}, nil
}

func (e *empty) Equal(other driver.Pagination) bool {
	_, ok := other.(*empty)
	return ok
}

func (k *empty) Serialize() ([]byte, error) {
	return []byte{}, nil
}

func EmptyFromRaw(raw []byte) (*empty, error) {
	return &empty{}, nil
}

type none struct{}

// None returns a pagination instance that will force a query to return everything in one shot
func None() *none {
	return &none{}
}

func (p *none) Prev() (driver.Pagination, error) {
	return Empty(), nil
}

func (p *none) Next() (driver.Pagination, error) {
	return Empty(), nil
}

func (e *none) Equal(other driver.Pagination) bool {
	_, ok := other.(*none)
	return ok
}

func (k *none) Serialize() ([]byte, error) {
	return []byte{}, nil
}

func NoneFromRaw(raw []byte) (*none, error) {
	return &none{}, nil
}
