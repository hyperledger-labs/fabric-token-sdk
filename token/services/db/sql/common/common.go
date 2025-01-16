/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
)

const (
	QueryLabel      tracing.LabelName = "query"
	ResultRowsLabel tracing.LabelName = "result_rows"
)

type Closer interface {
	Close() error
}

func Close(closer Closer) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		logger.Errorf("failed closing connection: %s", err)
	}
}
