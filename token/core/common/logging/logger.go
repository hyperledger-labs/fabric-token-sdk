/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"golang.org/x/exp/slices"
)

const loggerNameSeparator = "."

// Logger provides logging API
type Logger = logging.Logger

func DriverLogger(prefix string, networkID string, channel string, namespace string) Logger {
	return logging.MustGetLogger(loggerName(prefix, networkID, channel, namespace))
}

func DriverLoggerFromPP(prefix string, ppIdentifier string) Logger {
	return logging.MustGetLogger(loggerName(prefix, ppIdentifier))
}

func isEmptyString(s string) bool { return len(s) == 0 }

func loggerName(parts ...string) string {
	return strings.Join(slices.DeleteFunc(parts, isEmptyString), loggerNameSeparator)
}
