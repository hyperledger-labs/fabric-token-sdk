/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"strings"
)

const (
	SelectStatement         = `SELECT`
	SelectDistinctStatement = `SELECT DISTINCT`
	InsertStatement         = `INSERT INTO`
	UpdateStatement         = `UPDATE`
)

type Select struct {
	stmt    string
	columns []string
	from    []string
	where   string
	orderBy string
}

func NewSelect(columns ...string) *Select {
	return &Select{
		stmt:    SelectStatement,
		columns: columns,
	}
}

func NewSelectDistinct(columns ...string) *Select {
	return &Select{
		stmt:    SelectDistinctStatement,
		columns: columns,
	}
}

func (s *Select) From(tables ...string) *Select {
	s.from = tables
	return s
}

func (s *Select) Where(where string) *Select {
	s.where = where
	return s
}

func (s *Select) Compile() (string, error) {
	sb := new(strings.Builder)
	sb.WriteString(s.stmt)
	sb.WriteString(" ")
	if len(s.columns) > 0 {
		sb.WriteString(strings.Join(s.columns, ","))
		sb.WriteString(" ")
	}
	if len(s.from) > 0 {
		sb.WriteString("FROM ")
		sb.WriteString(strings.Join(s.from, " "))
		sb.WriteString(" ")
	}
	if len(s.where) > 0 {
		if !strings.HasPrefix(s.where, "WHERE") {
			sb.WriteString(" WHERE ")
		}
		sb.WriteString(s.where)
	}
	if len(s.orderBy) > 0 {
		if !strings.HasPrefix(s.orderBy, "ORDER BY") {
			sb.WriteString(" ORDER BY ")
		}
		sb.WriteString(s.orderBy)
	}
	return sb.String(), nil
}

func (s *Select) OrderBy(orderBy string) *Select {
	s.orderBy = orderBy
	return s
}

type Insert struct {
	stmt  string
	rows  string
	table string
}

func NewInsertInto(table string) *Insert {
	return &Insert{
		stmt:  InsertStatement,
		table: table,
	}
}

func (i *Insert) Rows(rows string) *Insert {
	i.rows = rows
	return i
}

func (i *Insert) Compile() (string, error) {
	sb := new(strings.Builder)
	sb.WriteString(i.stmt)
	sb.WriteString(" ")
	sb.WriteString(i.table)
	sb.WriteString(" ")
	if !strings.HasPrefix(i.rows, "(") {
		sb.WriteString("(")
	}
	sb.WriteString(i.rows)
	if !strings.HasSuffix(i.rows, ")") {
		sb.WriteString(")")
	}
	sb.WriteString(" ")

	// count number of rows
	splitRows := strings.Split(i.rows, ",")
	sb.WriteString("VALUES ")
	sb.WriteString("($1")
	for i := 2; i <= len(splitRows); i++ {
		sb.WriteString(fmt.Sprintf(", $%d", i))
	}
	sb.WriteString(")")
	return sb.String(), nil
}

type Update struct {
	stmt  string
	table string
	rows  string
	where string
}

func (u *Update) Set(rows string) *Update {
	u.rows = rows
	return u
}

func (u *Update) Where(where string) *Update {
	u.where = where
	return u
}

func (u *Update) Compile() (string, error) {
	counter := 1
	sb := new(strings.Builder)
	sb.WriteString(u.stmt)
	sb.WriteString(" ")
	sb.WriteString(u.table)
	sb.WriteString(" SET ")
	splitRows := strings.Split(u.rows, ",")
	for i, row := range splitRows {
		sb.WriteString(fmt.Sprintf("%s = $%d", row, counter))
		if i < len(splitRows)-1 {
			sb.WriteString(", ")
		}
		counter++
	}
	sb.WriteString(" ")

	if len(u.where) > 0 {
		if !strings.HasPrefix(u.where, "WHERE") {
			sb.WriteString(" WHERE ")
		}
		if !strings.Contains(u.where, "$") {
			splitWhere := strings.Split(u.where, ",")
			for i, row := range splitWhere {
				sb.WriteString(fmt.Sprintf("%s = $%d", row, counter))
				if i < len(splitWhere)-1 {
					sb.WriteString(" AND ")
				}
				counter++
			}
			sb.WriteString(" ")
		} else {
			sb.WriteString(u.where)
		}
	}
	return sb.String(), nil
}

func NewUpdate(table string) *Update {
	return &Update{
		stmt:  UpdateStatement,
		table: table,
	}
}
