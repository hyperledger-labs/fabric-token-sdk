/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package _insert

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
)

type onConflictSet struct {
	field common.FieldName
	value common.Param
}

func Set(field common.FieldName, value common.Param) OnConflict {
	return onConflictSet{
		field: field,
		value: value,
	}
}

func (o onConflictSet) WriteString(sb common.Builder) {
	sb.WriteSerializables(o.field).
		WriteString("=").
		WriteParam(o.value)
}

func Overwrite(field common.FieldName) OnConflict {
	return onConflictKeep{field: field}
}

type onConflictKeep struct{ field common.FieldName }

func (o onConflictKeep) WriteString(sb common.Builder) {
	sb.WriteSerializables(o.field).
		WriteString("=excluded.").
		WriteSerializables(o.field)
}
