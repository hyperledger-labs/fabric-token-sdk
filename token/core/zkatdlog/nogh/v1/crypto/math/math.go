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

// BaseElement is an interface for elements that belong to a mathematical curve.
type BaseElement interface {
	// CurveID returns the identifier of the curve this element belongs to.
	CurveID() mathlib.CurveID
}

// Element is an interface for curve elements that can also be checked for infinity.
type Element interface {
	BaseElement
	// IsInfinity returns true if the element is the point at infinity.
	IsInfinity() bool
}

// CheckElements validates a slice of elements against a curve ID and an expected length.
// It returns an error if the length is incorrect or if any element is invalid.
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

// CheckZrElements validates a slice of base elements against a curve ID and an expected length.
// It returns an error if the length is incorrect or if any element is invalid.
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

// CheckElement validates a single element: it must not be nil, must belong to the specified curve,
// and must not be the point at infinity.
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

// CheckBaseElement validates a single base element: it must not be nil and must belong
// to the specified curve.
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
