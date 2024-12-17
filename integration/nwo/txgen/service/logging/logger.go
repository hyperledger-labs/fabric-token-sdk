/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type Logger = logging.Logger

// MustGetLogger Get a logger.
func MustGetLogger(name string) Logger {
	return logging.MustGetLogger(name)
}
