/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package _insert

import (
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
)

type query struct {
	table          common2.TableName
	fields         []common2.FieldName
	rows           []common2.Tuple
	conflictFields []common2.FieldName
	onConflicts    []OnConflict
	ignoreConflict bool
}

func NewQuery() *query {
	return &query{}
}

func (q *query) Into(table common2.TableName) Query {
	q.table = table
	return q
}

func (q *query) Fields(fields ...common2.FieldName) fieldsQuery {
	q.fields = fields
	return q
}

func (q *query) Row(tuple ...common2.Param) fieldsQuery {
	if len(tuple) != len(q.fields) {
		panic("wrong length")
	}
	q.rows = append(q.rows, tuple)
	return q
}

func (q *query) Rows(tuples []common2.Tuple) fieldsQuery {
	for _, tuple := range tuples {
		q.Row(tuple...)
	}
	return q
}

func (q *query) OnConflict(fields []common2.FieldName, onConflicts ...OnConflict) onConflictQuery {
	if len(onConflicts) == 0 {
		panic("no strategy passed")
	}
	q.conflictFields = fields
	q.onConflicts = onConflicts
	return q
}

func (q *query) OnConflictDoNothing() onConflictQuery {
	q.ignoreConflict = true
	return q
}

func (q *query) Format() (string, []common2.Param) {
	sb := common2.NewBuilder()
	q.FormatTo(sb)
	return sb.Build()
}

func (q *query) FormatTo(sb common2.Builder) {
	sb.WriteString("INSERT INTO ").
		WriteString(string(q.table)).
		WriteString(" (").
		WriteSerializables(common2.ToSerializables(q.fields)...).
		WriteString(") VALUES ").
		WriteTuples(q.rows)

	if q.ignoreConflict {
		sb.WriteString(" ON CONFLICT DO NOTHING")
		return
	}
	if q.conflictFields != nil {
		sb.WriteString(" ON CONFLICT (").
			WriteSerializables(common2.ToSerializables(q.conflictFields)...).
			WriteString(") DO UPDATE SET ").
			WriteSerializables(common2.ToSerializables(q.onConflicts)...)
	}
}
