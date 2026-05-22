/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pagination

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
)

// ApplyToSquirrel applies a driver.Pagination to a squirrel SelectBuilder by adding
// the appropriate LIMIT, OFFSET, ORDER BY, and WHERE clauses.
// It lives in this package because it needs access to the unexported pagination types.
func ApplyToSquirrel(p driver.Pagination, sb sq.SelectBuilder) sq.SelectBuilder {
	switch pg := p.(type) {
	case nil:
		return sb

	case *none:
		return sb

	case *offset:
		sb = sb.Limit(uint64(pg.PageSize))
		if pg.Offset > 0 {
			sb = sb.Offset(uint64(pg.Offset))
		}
		return sb

	case *keyset[string, any]:
		return applyKeysetSquirrel(pg, sb)

	case *keyset[int, any]:
		return applyKeysetSquirrel(pg, sb)

	case *empty:
		return sb.Limit(0)

	default:
		panic(fmt.Sprintf("invalid pagination option %+v", p))
	}
}

func applyKeysetSquirrel[T comparable](pg *keyset[T, any], sb sq.SelectBuilder) sq.SelectBuilder {
	col := string(pg.SQLIDName)
	sb = sb.OrderBy(col + " ASC").Limit(uint64(pg.PageSize))
	if pg.FirstID != pg.nilElement() {
		sb = sb.Where(sq.Gt{col: pg.FirstID})
	} else if pg.Offset > 0 {
		sb = sb.Offset(uint64(pg.Offset))
	}
	return sb
}
