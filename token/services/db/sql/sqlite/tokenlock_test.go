/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"testing"
	"time"

	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	. "github.com/onsi/gomega"
)

func TestIsStale(t *testing.T) {
	var leaseExpiry time.Duration = 5
	ci := sqlite.NewConditionInterpreter()
	query, params := q.DeleteFrom("TokenLocks").
		Where(IsStale(common.TableName("TokenLocks"), common.TableName("Requests"), leaseExpiry)).
		Format(ci)

	expectedQuery := "DELETE FROM TokenLocks WHERE tx_id IN (SELECT TokenLocks.tx_id FROM TokenLocks JOIN Requests ON TokenLocks.tx_id = Requests.tx_id WHERE Requests.status IN ($1) OR TokenLocks.created_at < datetime('now', '-$2 seconds')"
	expectedParams := []common.Param{driver.Deleted, leaseExpiry.Seconds()}
	gt := NewGomegaWithT(t)
	gt.Expect(query).To(Equal(expectedQuery))
	gt.Expect(params).To(ConsistOf(expectedParams...))
}
