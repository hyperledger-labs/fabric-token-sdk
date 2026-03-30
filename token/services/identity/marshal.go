package identity

import (
	"errors"
	"math"
	"unsafe"
)

// DER universal tags
const (
	tagInteger     = byte(0x02)
	tagOctetString = byte(0x04)
	tagUTF8String  = byte(0x0C)
	tagSequence    = byte(0x30)
)

// Sentinel errors — no fmt.Errorf allocation on hot path
var (
	ErrTruncated     = errors.New("asn1: truncated data")
	ErrUnexpectedTag = errors.New("asn1: unexpected tag")
	ErrIntOverflow   = errors.New("asn1: integer overflows int32")
	ErrInvalidLen    = errors.New("asn1: invalid length encoding")
)

// Result holds the decoded payload. IsInt distinguishes the two variants.
// Data is a zero-copy sub-slice of the input — do not modify input while using it.
type Result struct {
	IsInt bool
	Int32 int32  // valid when IsInt == true
	Str   string // valid when IsInt == false
	Data  []byte // zero-copy reference into input
}

// Decode parses a DER SEQUENCE containing either
// [INTEGER, OCTET STRING] or [UTF8String, OCTET STRING].
func Decode(b []byte) (Result, error) {
	var r Result

	// Outer SEQUENCE
	if len(b) < 2 || b[0] != tagSequence {
		return r, ErrUnexpectedTag
	}
	_, pos, err := readLen(b, 1) // skip SEQUENCE length; we trust inner bounds checks
	if err != nil {
		return r, err
	}

	// Dispatch on first element's tag
	if pos >= len(b) {
		return r, ErrTruncated
	}
	switch b[pos] {
	case tagInteger:
		pos++
		l, np, err := readLen(b, pos)
		if err != nil {
			return r, err
		}
		if np+l > len(b) {
			return r, ErrTruncated
		}
		v, err := parseInt32(b[np : np+l])
		if err != nil {
			return r, err
		}
		r.IsInt = true
		r.Int32 = v
		pos = np + l

	case tagUTF8String:
		pos++
		l, np, err := readLen(b, pos)
		if err != nil {
			return r, err
		}
		if np+l > len(b) {
			return r, ErrTruncated
		}
		// unsafe.String: zero-copy conversion (Go 1.20+).
		// Safe as long as caller does not mutate b during Result lifetime.
		r.Str = unsafe.String(unsafe.SliceData(b[np:np+l]), l)
		pos = np + l

	default:
		return r, ErrUnexpectedTag
	}

	// OCTET STRING
	if pos >= len(b) || b[pos] != tagOctetString {
		return r, ErrUnexpectedTag
	}
	pos++
	l, np, err := readLen(b, pos)
	if err != nil {
		return r, err
	}
	if np+l > len(b) {
		return r, ErrTruncated
	}
	r.Data = b[np : np+l] // zero-copy
	return r, nil
}

// readLen decodes a DER length at b[pos]. Returns (length, nextPos, err).
func readLen(b []byte, pos int) (int, int, error) {
	if pos >= len(b) {
		return 0, 0, ErrTruncated
	}
	fb := b[pos]
	if fb < 0x80 { // short form
		return int(fb), pos + 1, nil
	}
	n := int(fb & 0x7F)
	if n == 0 || n > 4 || pos+1+n > len(b) { // cap at 4 bytes = 4 GiB
		return 0, 0, ErrInvalidLen
	}
	pos++
	l := 0
	for i := range n {
		l = l<<8 | int(b[pos+i])
	}
	return l, pos + n, nil
}

// parseInt32 decodes a DER big-endian signed integer into int32.
func parseInt32(b []byte) (int32, error) {
	if len(b) == 0 || len(b) > 5 {
		return 0, ErrIntOverflow
	}
	var v int64
	if b[0]&0x80 != 0 {
		v = -1 // sign-extend
	}
	for _, c := range b {
		v = v<<8 | int64(c)
	}
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, ErrIntOverflow
	}
	return int32(v), nil
}

// Encode serializes a Result back to DER for testing/interop.
func Encode(r Result) []byte {
	var first []byte
	if r.IsInt {
		first = appendTLV(nil, tagInteger, encodeInt32(r.Int32))
	} else {
		first = appendTLV(nil, tagUTF8String, []byte(r.Str))
	}
	body := append(first, appendTLV(nil, tagOctetString, r.Data)...)
	return appendTLV(nil, tagSequence, body)
}

func appendTLV(dst []byte, tag byte, val []byte) []byte {
	dst = append(dst, tag)
	l := len(val)
	switch {
	case l < 0x80:
		dst = append(dst, byte(l))
	case l < 0x100:
		dst = append(dst, 0x81, byte(l))
	default:
		dst = append(dst, 0x82, byte(l>>8), byte(l))
	}
	return append(dst, val...)
}

func encodeInt32(v int32) []byte {
	var b [4]byte
	b[0] = byte(v >> 24); b[1] = byte(v >> 16)
	b[2] = byte(v >> 8);  b[3] = byte(v)
	i := 0
	for i < 3 && b[i] == 0x00 && b[i+1]&0x80 == 0 { i++ }
	for i < 3 && b[i] == 0xFF && b[i+1]&0x80 != 0 { i++ }
	return b[i:]
}