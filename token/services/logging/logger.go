/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
)

const loggerNameSeparator = "."

// Logger provides logging API
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	IsEnabledFor(level zapcore.Level) bool
}

func MustGetLogger(loggerName string) Logger {
	return flogging.MustGetLogger(loggerName)
}

func DriverLogger(prefix string, networkID string, channel string, namespace string) Logger {
	return flogging.MustGetLogger(loggerName(prefix, networkID, channel, namespace))
}

func DeriveDriverLogger(logger Logger, prefix string, networkID string, channel string, namespace string) Logger {
	l, ok := logger.(*flogging.FabricLogger)
	if !ok {
		panic("invalid logger")
	}
	return l.Named(loggerName(prefix, networkID, channel, namespace))
}

func DriverLoggerFromPP(prefix string, ppIdentifier string) Logger {
	return flogging.MustGetLogger(loggerName(prefix, ppIdentifier))
}

func isEmptyString(s string) bool { return len(s) == 0 }

func loggerName(parts ...string) string {
	return strings.Join(slices.DeleteFunc(parts, isEmptyString), loggerNameSeparator)
}
