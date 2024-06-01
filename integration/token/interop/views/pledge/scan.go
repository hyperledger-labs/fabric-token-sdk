/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
)

// Scan contains the input information for a scan of a matching pledge id
type Scan struct {
	// TMSID identifies the TMS to use to perform the token operation
	TMSID token.TMSID
	// PlegdeID is the identifier to use in the scan
	PledgeID string
	// Timeout of the scan
	Timeout time.Duration
	// StartingTransactionID  is the transaction id from which to start the scan.
	// If empty, the scan starts from the genesis block
	StartingTransactionID string
}

type ScanView struct {
	*Scan
}

func (s *ScanView) Call(context view.Context) (interface{}, error) {
	b, err := pledge.IDExists(
		context,
		s.PledgeID,
		s.Timeout,
		token.WithTMSID(s.TMSID),
		pledge.WithStartingTransaction(s.StartingTransactionID),
	)
	assert.NoError(err, "failed to scan for pledge id")
	return b, nil
}

type ScanViewFactory struct{}

func (s *ScanViewFactory) NewView(in []byte) (view.View, error) {
	f := &ScanView{Scan: &Scan{}}
	err := json.Unmarshal(in, f.Scan)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
