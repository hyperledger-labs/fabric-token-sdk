/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

// Bind writes a new query parameter placeholder.
func Bind(v Param) Serializable {
	return bindParam{v: v}
}

type bindParam struct {
	v Param
}

func (b bindParam) WriteString(sb Builder) {
	sb.WriteParam(b.v)
}

// Ref references an already-bound parameter by its 1-based index.
func Ref(n int) Serializable {
	return paramRef{n: n}
}

type paramRef struct {
	n int
}

func (r paramRef) WriteString(sb Builder) {
	sb.WriteParamRef(r.n)
}

// IntervalAfterNow writes NOW() + $<ref>::interval using an existing bound parameter.
func IntervalAfterNow(ref int) Serializable {
	return intervalAfterNow{ref: ref}
}

type intervalAfterNow struct {
	ref int
}

func (i intervalAfterNow) WriteString(sb Builder) {
	sb.WriteString("NOW() + ").WriteParamRef(i.ref).WriteString("::interval")
}
