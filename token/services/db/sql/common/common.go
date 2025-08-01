/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "context"

type Closer interface {
	Close() error
}

func Close(closer Closer) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		logger.ErrorfContext(context.Background(), "failed closing connection: %s", err)
	}
}
