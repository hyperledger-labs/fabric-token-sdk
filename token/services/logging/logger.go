/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"strings"

	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
)

const loggerNameSeparator = "."

// Logger provides logging API
type Logger = logging.Logger

func MustGetLogger(params ...string) Logger {
	return utils.MustGet(GetLogger(params...))
}

func GetLogger(params ...string) (Logger, error) {
	return logging.GetLoggerWithReplacements(map[string]string{"github.com.hyperledger-labs.fabric-token-sdk.token": "fts"}, params)
}
func DriverLogger(prefix string, networkID string, channel string, namespace string) Logger {
	return logging.MustGetLogger(loggerName(prefix, networkID, channel, namespace))
}

func DeriveDriverLogger(logger Logger, prefix string, networkID string, channel string, namespace string) Logger {
	return logger.Named(loggerName(prefix, networkID, channel, namespace))
}

func DriverLoggerFromPP(prefix string, id string) Logger {
	return logging.MustGetLogger(loggerName(prefix, id))
}

func isEmptyString(s string) bool { return len(s) == 0 }

func loggerName(parts ...string) string {
	return strings.Join(slices.DeleteFunc(parts, isEmptyString), loggerNameSeparator)
}
