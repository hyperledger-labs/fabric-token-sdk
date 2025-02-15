/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ProtoSource[T any] interface {
	ToProtos() (*T, error)
}

type ProtoDestination[T any] interface {
	FromProtos(*T) error
}

func ToProtosSlice[T any, S ProtoSource[T]](s []S) ([]*T, error) {
	if len(s) == 0 {
		return nil, nil
	}
	res := make([]*T, len(s))
	var err error
	for i, x := range s {
		if IsNil(x) {
			res[i] = nil
			continue
		}

		res[i], err = x.ToProtos()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func ToProtosSliceFunc[T any, S any](s []S, convert func(S) (*T, error)) ([]*T, error) {
	if len(s) == 0 {
		return nil, nil
	}
	res := make([]*T, len(s))
	var err error
	for i, x := range s {
		if IsNil(x) {
			res[i] = nil
			continue
		}

		res[i], err = convert(x)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func FromProtosSlice[T any, S ProtoDestination[T]](t []*T, s []S) error {
	var err error
	for i, x := range s {
		if t[i] == nil {
			var a S
			s[i] = a
			continue
		}
		err = x.FromProtos(t[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func FromProtosSliceFunc[T any, S any](s []S, convert func(S) (*T, error)) ([]*T, error) {
	var err error
	res := make([]*T, len(s))
	for i, x := range s {
		res[i], err = convert(x)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func FromProtosSliceFunc2[T any, S any](s []S, convert func(S) (T, error)) ([]T, error) {
	if len(s) == 0 {
		return nil, nil
	}
	var err error
	res := make([]T, len(s))
	for i, x := range s {
		res[i], err = convert(x)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func ToActionSlice(actionType request.ActionType, actions [][]byte) []*request.Action {
	res := make([]*request.Action, len(actions))
	for i, action := range actions {
		res[i] = &request.Action{
			Type: actionType,
			Raw:  action,
		}
	}
	return res
}

func ToSignatureSlice(signatures [][]byte) []*request.Signature {
	res := make([]*request.Signature, len(signatures))
	for i, signature := range signatures {
		res[i] = &request.Signature{
			Raw: signature,
		}
	}
	return res
}

func ToTokenID(id *token.ID) (*request.TokenID, error) {
	if id == nil {
		return nil, nil
	}
	return &request.TokenID{
		TxId:  id.TxId,
		Index: id.Index,
	}, nil
}

func IsNil[T any](value T) bool {
	// Use reflection to check if the value is nil
	v := reflect.ValueOf(value)
	return (v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice || v.Kind() == reflect.Map || v.Kind() == reflect.Chan || v.Kind() == reflect.Func || v.Kind() == reflect.Interface) && v.IsNil()
}

func GenericSliceOfPointers[T any](size int) []*T {
	slice := make([]*T, size)
	for i := range slice {
		var zero T
		slice[i] = &zero
	}
	return slice
}
