/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

type Field Serializable

// FieldName is the name of the DB column
type FieldName string

func (n FieldName) WriteString(b Builder) {
	b.WriteString(string(n))
}

type field struct {
	table *aliasedTable
	name  FieldName
}

func (f field) WriteString(b Builder) {
	if f.table != nil {
		b.WriteString(string(f.table.Alias())).WriteRune('.')
	}
	b.WriteString(string(f.name))
}
