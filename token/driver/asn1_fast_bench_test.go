/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"
	"testing"
)

// BenchmarkFastMarshalTokenRequestForSigning_Small benchmarks fast marshaller with small data
func BenchmarkFastMarshalTokenRequestForSigning_Small(b *testing.B) {
	issues := [][]byte{[]byte("issue1")}
	transfers := [][]byte{[]byte("transfer1")}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalTokenRequestForSigning(issues, transfers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdMarshalTokenRequestForSigning_Small benchmarks standard ASN.1 with small data
func BenchmarkStdMarshalTokenRequestForSigning_Small(b *testing.B) {
	type tokenRequestForSigning struct {
		Issues    [][]byte
		Transfers [][]byte
	}
	req := tokenRequestForSigning{
		Issues:    [][]byte{[]byte("issue1")},
		Transfers: [][]byte{[]byte("transfer1")},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastMarshalTokenRequestForSigning_Medium benchmarks fast marshaller with medium data
func BenchmarkFastMarshalTokenRequestForSigning_Medium(b *testing.B) {
	issues := [][]byte{
		make([]byte, 100),
		make([]byte, 200),
		make([]byte, 150),
	}
	transfers := [][]byte{
		make([]byte, 180),
		make([]byte, 220),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalTokenRequestForSigning(issues, transfers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdMarshalTokenRequestForSigning_Medium benchmarks standard ASN.1 with medium data
func BenchmarkStdMarshalTokenRequestForSigning_Medium(b *testing.B) {
	type tokenRequestForSigning struct {
		Issues    [][]byte
		Transfers [][]byte
	}
	req := tokenRequestForSigning{
		Issues: [][]byte{
			make([]byte, 100),
			make([]byte, 200),
			make([]byte, 150),
		},
		Transfers: [][]byte{
			make([]byte, 180),
			make([]byte, 220),
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastMarshalTokenRequestForSigning_Large benchmarks fast marshaller with large data
func BenchmarkFastMarshalTokenRequestForSigning_Large(b *testing.B) {
	issues := [][]byte{
		make([]byte, 5000),
		make([]byte, 8000),
		make([]byte, 6000),
		make([]byte, 7000),
	}
	transfers := [][]byte{
		make([]byte, 4000),
		make([]byte, 9000),
		make([]byte, 5500),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalTokenRequestForSigning(issues, transfers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdMarshalTokenRequestForSigning_Large benchmarks standard ASN.1 with large data
func BenchmarkStdMarshalTokenRequestForSigning_Large(b *testing.B) {
	type tokenRequestForSigning struct {
		Issues    [][]byte
		Transfers [][]byte
	}
	req := tokenRequestForSigning{
		Issues: [][]byte{
			make([]byte, 5000),
			make([]byte, 8000),
			make([]byte, 6000),
			make([]byte, 7000),
		},
		Transfers: [][]byte{
			make([]byte, 4000),
			make([]byte, 9000),
			make([]byte, 5500),
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastMarshalSignatureMessageV2_Small benchmarks fast marshaller with small signature message
func BenchmarkFastMarshalSignatureMessageV2_Small(b *testing.B) {
	request := []byte("small-request-data")
	anchor := []byte("anchor")

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalSignatureMessageV2(request, anchor)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdMarshalSignatureMessageV2_Small benchmarks standard ASN.1 with small signature message
func BenchmarkStdMarshalSignatureMessageV2_Small(b *testing.B) {
	type signatureMessage struct {
		Request []byte
		Anchor  []byte
	}
	msg := signatureMessage{
		Request: []byte("small-request-data"),
		Anchor:  []byte("anchor"),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastMarshalSignatureMessageV2_Medium benchmarks fast marshaller with medium signature message
func BenchmarkFastMarshalSignatureMessageV2_Medium(b *testing.B) {
	request := make([]byte, 1000)
	anchor := make([]byte, 64)

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalSignatureMessageV2(request, anchor)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdMarshalSignatureMessageV2_Medium benchmarks standard ASN.1 with medium signature message
func BenchmarkStdMarshalSignatureMessageV2_Medium(b *testing.B) {
	type signatureMessage struct {
		Request []byte
		Anchor  []byte
	}
	msg := signatureMessage{
		Request: make([]byte, 1000),
		Anchor:  make([]byte, 64),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastMarshalSignatureMessageV2_Large benchmarks fast marshaller with large signature message
func BenchmarkFastMarshalSignatureMessageV2_Large(b *testing.B) {
	request := make([]byte, 50000)
	anchor := make([]byte, 128)

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalSignatureMessageV2(request, anchor)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdMarshalSignatureMessageV2_Large benchmarks standard ASN.1 with large signature message
func BenchmarkStdMarshalSignatureMessageV2_Large(b *testing.B) {
	type signatureMessage struct {
		Request []byte
		Anchor  []byte
	}
	msg := signatureMessage{
		Request: make([]byte, 50000),
		Anchor:  make([]byte, 128),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalToMessageToSignV2_Complete benchmarks the complete V2 marshalling flow
func BenchmarkMarshalToMessageToSignV2_Complete(b *testing.B) {
	tr := &TokenRequest{
		Issues: [][]byte{
			make([]byte, 500),
			make([]byte, 800),
		},
		Transfers: [][]byte{
			make([]byte, 600),
		},
	}
	anchor := []byte("test-anchor-data")

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := tr.marshalToMessageToSignV2(anchor)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalToMessageToSignV1_Complete benchmarks the V1 marshalling flow for comparison
func BenchmarkMarshalToMessageToSignV1_Complete(b *testing.B) {
	tr := &TokenRequest{
		Issues: [][]byte{
			make([]byte, 500),
			make([]byte, 800),
		},
		Transfers: [][]byte{
			make([]byte, 600),
		},
	}
	anchor := []byte("test-anchor-data")

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := tr.marshalToMessageToSignV1(anchor)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManySmallItems benchmarks performance with many small items
func BenchmarkManySmallItems_Fast(b *testing.B) {
	issues := make([][]byte, 50)
	transfers := make([][]byte, 50)
	for i := range issues {
		issues[i] = []byte{byte(i)}
		transfers[i] = []byte{byte(i + 50)}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := fastMarshalTokenRequestForSigning(issues, transfers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManySmallItems_Std benchmarks standard ASN.1 with many small items
func BenchmarkManySmallItems_Std(b *testing.B) {
	type tokenRequestForSigning struct {
		Issues    [][]byte
		Transfers [][]byte
	}
	issues := make([][]byte, 50)
	transfers := make([][]byte, 50)
	for i := range issues {
		issues[i] = []byte{byte(i)}
		transfers[i] = []byte{byte(i + 50)}
	}
	req := tokenRequestForSigning{
		Issues:    issues,
		Transfers: transfers,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := asn1.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Made with Bob
