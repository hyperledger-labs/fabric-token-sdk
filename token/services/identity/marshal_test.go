package identity_test

import (
	"bytes"
	"encoding/asn1"
	"math"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
)

// ---------------------------------------------------------------------------
// Reference structs — used only to drive encoding/asn1.Marshal / Unmarshal.
// The `asn1:"utf8"` tag makes Marshal emit tag 0x0C (UTF8String), which
// matches what our decoder expects.
// ---------------------------------------------------------------------------

type refInt struct {
	Value int32
	Data  []byte
}

type refStr struct {
	Value string `asn1:"utf8"`
	Data  []byte
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testing.TB lets the same helpers work for both *testing.T and *testing.B.
func marshalInt(tb testing.TB, v int32, data []byte) []byte {
	tb.Helper()
	b, err := asn1.Marshal(refInt{Value: v, Data: data})
	if err != nil {
		tb.Fatalf("asn1.Marshal(int): %v", err)
	}
	return b
}

func marshalStr(tb testing.TB, s string, data []byte) []byte {
	tb.Helper()
	b, err := asn1.Marshal(refStr{Value: s, Data: data})
	if err != nil {
		tb.Fatalf("asn1.Marshal(str): %v", err)
	}
	return b
}

// dataEq treats nil and []byte{} as equal — DER OCTET STRING of length 0
// decodes to a non-nil empty slice even when the original was nil.
func dataEq(a, b []byte) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return bytes.Equal(a, b)
}

// ---------------------------------------------------------------------------
// Integer variant — decode bytes produced by encoding/asn1
// ---------------------------------------------------------------------------

func TestDecodeIntVariant(t *testing.T) {
	tests := []struct {
		name  string
		value int32
		data  []byte
	}{
		{"zero/nil-data",         0,              nil},
		{"zero/empty-data",       0,              []byte{}},
		{"positive/small",        42,             []byte{0x01, 0x02, 0x03}},
		{"positive/max-int32",    math.MaxInt32,  []byte("hello")},
		{"negative/-1",           -1,             []byte{0xFF}},
		{"negative/min-int32",    math.MinInt32,  []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{"large-data/300-bytes",  1,              bytes.Repeat([]byte{0xAB}, 300)},
		{"single-byte-data",      100,            []byte{0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := marshalInt(t, tt.value, tt.data)

			got, err := identity.Decode(enc)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if !got.IsInt {
				t.Fatal("IsInt = false, want true")
			}
			if got.Int32 != tt.value {
				t.Errorf("Int32 = %d, want %d", got.Int32, tt.value)
			}
			if !dataEq(got.Data, tt.data) {
				t.Errorf("Data = %x, want %x", got.Data, tt.data)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// String variant — decode bytes produced by encoding/asn1
// ---------------------------------------------------------------------------

func TestDecodeStrVariant(t *testing.T) {
	tests := []struct {
		name  string
		value string
		data  []byte
	}{
		{"empty-str/nil-data",  "",                                      nil},
		{"ascii",               "hello",                                  []byte{0x01, 0x02, 0x03}},
		{"unicode/latin",       "héllo wörld",                           []byte{0xAA, 0xBB}},
		{"unicode/cjk",         "日本語",                                 []byte{0xCA, 0xFE}},
		{"unicode/mixed",       "Go: 高速 ASN.1 🚀",                     []byte("payload")},
		{"empty-data",          "test",                                   []byte{}},
		{"long-string/200",     string(bytes.Repeat([]byte("x"), 200)),  []byte{0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := marshalStr(t, tt.value, tt.data)

			got, err := identity.Decode(enc)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got.IsInt {
				t.Fatal("IsInt = true, want false")
			}
			if got.Str != tt.value {
				t.Errorf("Str = %q, want %q", got.Str, tt.value)
			}
			if !dataEq(got.Data, tt.data) {
				t.Errorf("Data = %x, want %x", got.Data, tt.data)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Round-trip: our Encode → our Decode
// ---------------------------------------------------------------------------

func TestRoundTripInt(t *testing.T) {
	cases := []struct {
		v int32
		d []byte
	}{
		{0, nil},
		{42, []byte("round-trip payload")},
		{math.MaxInt32, []byte{0xFF, 0x00}},
		{math.MinInt32, []byte{0x01}},
		{-100, bytes.Repeat([]byte{0x55}, 128)},
	}
	for _, c := range cases {
		r := identity.Result{IsInt: true, Int32: c.v, Data: c.d}
		enc := identity.Encode(r)
		got, err := identity.Decode(enc)
		if err != nil {
			t.Fatalf("Decode(Encode(int32(%d))): %v", c.v, err)
		}
		if !got.IsInt || got.Int32 != c.v || !dataEq(got.Data, c.d) {
			t.Errorf("round-trip mismatch: got {IsInt:%v Int32:%d Data:%x}, want {true %d %x}",
				got.IsInt, got.Int32, got.Data, c.v, c.d)
		}
	}
}

func TestRoundTripStr(t *testing.T) {
	cases := []struct {
		s string
		d []byte
	}{
		{"", nil},
		{"hello", []byte{0x01, 0x02}},
		{"héllo 日本語 🚀", []byte("data")},
	}
	for _, c := range cases {
		r := identity.Result{IsInt: false, Str: c.s, Data: c.d}
		enc := identity.Encode(r)
		got, err := identity.Decode(enc)
		if err != nil {
			t.Fatalf("Decode(Encode(str %q)): %v", c.s, err)
		}
		if got.IsInt || got.Str != c.s || !dataEq(got.Data, c.d) {
			t.Errorf("round-trip mismatch: got {IsInt:%v Str:%q Data:%x}, want {false %q %x}",
				got.IsInt, got.Str, got.Data, c.s, c.d)
		}
	}
}

// ---------------------------------------------------------------------------
// Cross-validation: our Encode → encoding/asn1.Unmarshal
// Verifies our encoder produces canonical DER that the stdlib accepts.
// ---------------------------------------------------------------------------

func TestEncodeProducesValidDER_Int(t *testing.T) {
	cases := []struct {
		v int32
		d []byte
	}{
		{42, []byte("data")},
		{math.MaxInt32, []byte{0x01}},
		{math.MinInt32, []byte{0xFF}},
		{-1, []byte("neg")},
		{0, []byte{}},
	}
	for _, c := range cases {
		enc := identity.Encode(identity.Result{IsInt: true, Int32: c.v, Data: c.d})
		var dst refInt
		rest, err := asn1.Unmarshal(enc, &dst)
		if err != nil {
			t.Errorf("asn1.Unmarshal(Encode(int32(%d))) = %v", c.v, err)
			continue
		}
		if len(rest) != 0 {
			t.Errorf("trailing bytes after Unmarshal: %x", rest)
		}
		if dst.Value != c.v {
			t.Errorf("Value = %d, want %d", dst.Value, c.v)
		}
		if !dataEq(dst.Data, c.d) {
			t.Errorf("Data mismatch: got %x, want %x", dst.Data, c.d)
		}
	}
}

func TestEncodeProducesValidDER_Str(t *testing.T) {
	cases := []struct {
		s string
		d []byte
	}{
		{"hello", []byte("data")},
		{"unicode 日本", []byte{0x01}},
		{"", []byte{0xAB}},
	}
	for _, c := range cases {
		enc := identity.Encode(identity.Result{IsInt: false, Str: c.s, Data: c.d})
		var dst refStr
		rest, err := asn1.Unmarshal(enc, &dst)
		if err != nil {
			t.Errorf("asn1.Unmarshal(Encode(str %q)) = %v", c.s, err)
			continue
		}
		if len(rest) != 0 {
			t.Errorf("trailing bytes after Unmarshal: %x", rest)
		}
		if dst.Value != c.s {
			t.Errorf("Value = %q, want %q", dst.Value, c.s)
		}
		if !dataEq(dst.Data, c.d) {
			t.Errorf("Data mismatch: got %x, want %x", dst.Data, c.d)
		}
	}
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestDecodeErrors(t *testing.T) {
	// Build a valid int-variant encoding to use as a base for mutations.
	validInt := marshalInt(t, 42, []byte{0x01})

	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{
			"empty input",
			[]byte{},
			identity.ErrUnexpectedTag,
		},
		{
			"wrong outer tag — OCTET STRING instead of SEQUENCE",
			[]byte{0x04, 0x03, 0x01, 0x02, 0x03},
			identity.ErrUnexpectedTag,
		},
		{
			"unsupported first field tag — BOOLEAN (0x01)",
			// SEQUENCE { BOOLEAN(true), OCTET STRING }
			[]byte{0x30, 0x06, 0x01, 0x01, 0xFF, 0x04, 0x01, 0x00},
			identity.ErrUnexpectedTag,
		},
		{
			"truncated — only outer tag, no length byte",
			[]byte{0x30},
			identity.ErrTruncated,
		},
		{
			"integer present but no OCTET STRING follows",
			// SEQUENCE { INTEGER(1) } — missing OCTET STRING
			func() []byte {
				b, _ := asn1.Marshal(struct{ V int32 }{1})
				return b
			}(),
			identity.ErrUnexpectedTag,
		},
		{
			"integer body truncated",
			// SEQUENCE { INTEGER claims 4 bytes but only 1 byte present }
			[]byte{0x30, 0x04, 0x02, 0x04, 0x01},
			identity.ErrTruncated,
		},
		{
			"valid prefix with junk outer tag copy",
			// Flip first byte of a known-good encoding
			func() []byte {
				b := make([]byte, len(validInt))
				copy(b, validInt)
				b[0] = 0x01
				return b
			}(),
			identity.ErrUnexpectedTag,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := identity.Decode(tt.input)
			if err != tt.wantErr {
				t.Errorf("Decode() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks — fast decoder vs encoding/asn1 stdlib
// ---------------------------------------------------------------------------

// Package-level sink prevents the compiler from optimising away benchmark calls.
var sink identity.Result

func BenchmarkDecodeInt_Fast(b *testing.B) {
	enc := marshalInt(b, 42, []byte("benchmark payload 1234"))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var err error
		sink, err = identity.Decode(enc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeStr_Fast(b *testing.B) {
	enc := marshalStr(b, "benchmark string", []byte("benchmark payload 1234"))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var err error
		sink, err = identity.Decode(enc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeInt_Stdlib(b *testing.B) {
	enc := marshalInt(b, 42, []byte("benchmark payload 1234"))
	b.ResetTimer()
	b.ReportAllocs()
	var dst refInt
	for i := 0; i < b.N; i++ {
		if _, err := asn1.Unmarshal(enc, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeStr_Stdlib(b *testing.B) {
	enc := marshalStr(b, "benchmark string", []byte("benchmark payload 1234"))
	b.ResetTimer()
	b.ReportAllocs()
	var dst refStr
	for i := 0; i < b.N; i++ {
		if _, err := asn1.Unmarshal(enc, &dst); err != nil {
			b.Fatal(err)
		}
	}
}