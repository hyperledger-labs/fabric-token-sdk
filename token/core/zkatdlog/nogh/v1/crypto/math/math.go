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
		return errors.Errorf("element is nil")
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
		return errors.Errorf("element is nil")
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

func InnerProduct(left []*mathlib.Zr, right []*mathlib.Zr, c *mathlib.Curve) *mathlib.Zr {
	return c.ModAddMul(left, right, c.GroupOrder)
}

// BatchInverse computes the entry-wise modular inverse of elems using
// Montgomery's trick: a single InvModOrder call plus O(n) multiplications.
// todo! Perhaps this can be added to mathlib.
func BatchInverse(elems []*mathlib.Zr, curve *mathlib.Curve) []*mathlib.Zr {
	n := len(elems)
	if n == 0 {
		return nil
	}

	inv := make([]*mathlib.Zr, n)

	// Forward pass: build prefix products
	// prefixProd[i] = elems[0] * elems[1] * ... * elems[i]
	prefixProd := make([]*mathlib.Zr, n)
	prefixProd[0] = elems[0]
	for i := 1; i < n; i++ {
		prefixProd[i] = curve.ModMul(prefixProd[i-1], elems[i], curve.GroupOrder)
	}

	// Single inversion of the total product
	acc := prefixProd[n-1]
	acc.InvModOrder()

	// Backward pass: extract individual inverses
	for i := n - 1; i >= 1; i-- {
		inv[i] = curve.ModMul(prefixProd[i-1], acc, curve.GroupOrder)
		acc = curve.ModMul(acc, elems[i], curve.GroupOrder)
	}
	inv[0] = acc

	return inv
}
