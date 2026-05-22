/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package _update

import (
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
	cond2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/cond"
)

type set struct {
	field common2.FieldName
	value common2.Param
}

func (s set) WriteString(sb common2.Builder) {
	sb.WriteSerializables(s.field).WriteString(" = ").WriteParam(s.value)
}

type query struct {
	table common2.TableName
	sets  []set
	where cond2.Condition
}

func NewQuery() *query {
	return &query{}
}

func (q *query) Update(t common2.TableName) Query {
	q.table = t
	return q
}

func (q *query) Set(field common2.FieldName, value common2.Param) setQuery {
	q.sets = append(q.sets, set{field: field, value: value})
	return q
}

func (q *query) Where(where cond2.Condition) whereQuery {
	q.where = where
	return q
}

func (q *query) Format(ci common2.CondInterpreter) (string, []common2.Param) {
	sb := common2.NewBuilder()
	q.FormatTo(ci, sb)
	return sb.Build()
}

func (q *query) FormatTo(ci common2.CondInterpreter, sb common2.Builder) {
	sb.WriteString("UPDATE ").
		WriteString(string(q.table)).
		WriteString(" SET ").
		WriteSerializables(common2.ToSerializables(q.sets)...)

	if q.where != nil && q.where != cond2.AlwaysTrue {
		sb.WriteString(" WHERE ").WriteConditionSerializable(q.where, ci)
	}
}
