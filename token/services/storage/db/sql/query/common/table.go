/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

type TableName string

// TODO: Support new alias per table in case a table is selected twice in the same query (for different joins)
func NewTable(name TableName) aliasedTable {
	return NewAliasedTable(name, TableAlias(name))
}

func NewAliasedTable(name TableName, alias TableAlias) aliasedTable {
	return aliasedTable{
		name:  name,
		alias: alias,
	}
}

type Table interface {
	JoinedTable
	Field(name FieldName) Field
}

type JoinedTable interface {
	ConditionSerializable
	JoinAs(typ JoinType, other Table, ons ConditionSerializable) JoinedTable
	Join(Table, ConditionSerializable) JoinedTable
}

type TableAlias string

type aliasedTable struct {
	name  TableName
	alias TableAlias
}

func (a aliasedTable) WriteString(_ CondInterpreter, sb Builder) {
	sb.WriteString(string(a.name))
	if len(a.alias) > 0 && string(a.alias) != string(a.name) {
		sb.WriteString(" AS ").WriteString(string(a.alias))
	}
}

func (a aliasedTable) Field(name FieldName) Field {
	return field{table: &a, name: name}
}

func (a aliasedTable) Alias() TableAlias { return a.alias }

func (a aliasedTable) JoinAs(typ JoinType, other Table, ons ConditionSerializable) JoinedTable {
	return joinedTable{
		types:      []JoinType{typ},
		tables:     []Table{a, other},
		conditions: []ConditionSerializable{ons},
	}
}

func (a aliasedTable) Join(other Table, ons ConditionSerializable) JoinedTable {
	return a.JoinAs(Left, other, ons)
}

type JoinType int

const (
	Left JoinType = iota
	Right
	Full
	Inner
	Cross
)

type joinedTable struct {
	types      []JoinType
	tables     []Table
	conditions []ConditionSerializable
}

func (t joinedTable) JoinAs(typ JoinType, other Table, ons ConditionSerializable) JoinedTable {
	t.types = append(t.types, typ)
	t.tables = append(t.tables, other)
	t.conditions = append(t.conditions, ons)
	return t
}

func (t joinedTable) Join(other Table, ons ConditionSerializable) JoinedTable {
	return t.JoinAs(Left, other, ons)
}

var joinTypeMap = map[JoinType]string{
	Left:  " LEFT JOIN ",  // same as JOIN, LEFT OUTER JOIN
	Right: " RIGHT JOIN ", // same as RIGHT OUTER JOIN
	Full:  " FULL JOIN ",  // same as FULL OUTER JOIN
	Inner: " INNER JOIN ",
	Cross: " CROSS JOIN ",
}

func (t joinedTable) WriteString(in CondInterpreter, sb Builder) {
	sb.WriteConditionSerializable(t.tables[0], in)
	for i, tt := range t.tables[1:] {
		sb.WriteString(joinTypeMap[t.types[i]]).
			WriteConditionSerializable(tt, in).
			WriteString(" ON ").
			WriteConditionSerializable(t.conditions[i], in)
	}
}
