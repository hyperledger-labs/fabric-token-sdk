/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package math

import (
	"reflect"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

type BaseElement interface {
	CurveID() mathlib.CurveID
}

type Element interface {
	BaseElement
	IsInfinity() bool
}

func CheckElements[E Element](elements []E, curveID mathlib.CurveID, length uint64) error {
	if uint64(len(elements)) != length {
		return errors.Errorf("length of elements does not match length of curveID")
	}
	for _, g1 := range elements {
		if err := CheckElement[E](g1, curveID); err != nil {
			return err
		}
	}
	return nil
}

func CheckZrElements[E BaseElement](elements []E, curveID mathlib.CurveID, length uint64) error {
	if uint64(len(elements)) != length {
		return errors.Errorf("length of elements does not match length of curveID")
	}
	for _, g1 := range elements {
		if err := CheckBaseElement[E](g1, curveID); err != nil {
			return err
		}
	}
	return nil
}

func CheckElement[E Element](element E, curveID mathlib.CurveID) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.Errorf("caught panic while checking element, err [%s]", e)
		}
	}()

	if isNilInterface(element) {
		return errors.Errorf("elememt is nil")
	}
	if element.CurveID() != curveID {
		return errors.Errorf("element curve must equal curve ID")
	}
	if element.IsInfinity() {
		return errors.New("element is infinity")
	}
	return nil
}

func CheckBaseElement[E BaseElement](element E, curveID mathlib.CurveID) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.Errorf("caught panic while checking element, err [%s]", e)
		}
	}()

	if isNilInterface(element) {
		return errors.Errorf("elememt is nil")
	}
	if element.CurveID() != curveID {
		return errors.Errorf("element curve must equal curve ID")
	}
	return nil
}

func isNilInterface(i interface{}) bool {
	if i == nil {
		return true
	}
	rv := reflect.ValueOf(i)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}
