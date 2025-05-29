/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"testing"
	"time"

	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	. "github.com/onsi/gomega"
)

func TestIsStale(t *testing.T) {
	RegisterTestingT(t)

	query, args := q.DeleteFrom("TokenLocks").
		Where(IsStale("TokenLocks", "Requests", 5*time.Second)).
		Format(sqlite.NewConditionInterpreter())

	Expect(query).To(Equal("DELETE FROM TokenLocks WHERE tx_id IN (" +
		"SELECT tl.tx_id " +
		"FROM TokenLocks AS tl " +
		"LEFT JOIN Requests AS tr " +
		"ON tl.tx_id = tr.tx_id " +
		"WHERE (tr.status = $1) OR (tl.created_at < datetime('now', '-5 seconds'))" +
		")"))
	Expect(args).To(ConsistOf(driver.Deleted))
}
