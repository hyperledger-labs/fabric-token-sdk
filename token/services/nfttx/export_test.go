/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

func NewTestQueryExecutor(s interface{}, v interface{}, p uint64) *QueryExecutor {
	return &QueryExecutor{
		selector:  s.(selector),
		vault:     v.(vault),
		precision: p,
	}
}
