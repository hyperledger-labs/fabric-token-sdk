/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"math/big"
	"testing"
)

func TestBigIntScan_Nil(t *testing.T) {
	var b BigInt
	if err := b.Scan(nil); err != nil {
		t.Fatalf("expected no error for nil, got %v", err)
	}
	if b.Sign() != 0 {
		t.Fatalf("expected 0 for nil, got %s", b.String())
	}
}

func TestBigIntScan_Bytes(t *testing.T) {
	var b BigInt
	if err := b.Scan([]byte("12345678901234567890")); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected, _ := new(big.Int).SetString("12345678901234567890", 10)
	if b.Cmp(expected) != 0 {
		t.Fatalf("expected %s, got %s", expected.String(), b.String())
	}
}

func TestBigIntScan_String(t *testing.T) {
	var b BigInt
	if err := b.Scan("98765432109876543210"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected, _ := new(big.Int).SetString("98765432109876543210", 10)
	if b.Cmp(expected) != 0 {
		t.Fatalf("expected %s, got %s", expected.String(), b.String())
	}
}

func TestBigIntScan_Int64(t *testing.T) {
	var b BigInt
	if err := b.Scan(int64(42)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.Int64() != 42 {
		t.Fatalf("expected 42, got %s", b.String())
	}
}

func TestBigIntScan_InvalidType(t *testing.T) {
	var b BigInt
	if err := b.Scan(3.14); err == nil {
		t.Fatal("expected error for float64 type, got nil")
	}
}

func TestBigIntScan_InvalidString(t *testing.T) {
	var b BigInt
	if err := b.Scan("not-a-number"); err == nil {
		t.Fatal("expected error for invalid string, got nil")
	}
}
