/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

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

func CloseFunc(closer func() error) {
	if closer == nil {
		return
	}
	if err := closer(); err != nil {
		logger.Errorf("failed closing connection: %s", err)
	}
}
