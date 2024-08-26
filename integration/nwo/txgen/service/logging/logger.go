/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"fmt"
	"os"
	"slices"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model"
	c "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model/constants"

	"github.com/hyperledger/fabric-lib-go/common/flogging"
)

// ILogger Interface for the logger used in the application.
type ILogger interface {
	DPanic(args ...interface{})
	DPanicf(template string, args ...interface{})
	DPanicw(msg string, kvPairs ...interface{})

	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Debugw(msg string, kvPairs ...interface{})

	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Errorw(msg string, kvPairs ...interface{})

	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
	Fatalw(msg string, kvPairs ...interface{})

	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Infow(msg string, kvPairs ...interface{})

	Panic(args ...interface{})
	Panicf(template string, args ...interface{})
	Panicw(msg string, kvPairs ...interface{})

	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Warnw(msg string, kvPairs ...interface{})
	Warning(args ...interface{})
	Warningf(template string, args ...interface{})
}

// Logger Global logger to use where no local logger is available.
var Logger ILogger

// InitializeLogger Initialize a logger for the application.
func InitializeLogger(config model.AppConfig) {
	// Use fabric logger
	initializeLogger(config)

	// set global logger
	Logger = MustGetLogger("E2E_Global_Log")
}

// MustGetLogger Get a logger.
func MustGetLogger(name string) ILogger {
	return flogging.MustGetLogger(name)
}

func getSupportedLevels() []model.LogLevel {
	return []model.LogLevel{
		c.DEBUG,
		c.INFO,
		c.WARN,
		c.ERROR,
		c.FATAL,
	}
}

// initializeLogger Initialize the Fabric Logger according to the params defined in config.yaml.
//
//	config model.Configuration The application configuration.
func initializeLogger(config model.AppConfig) {
	loggingLevel := config.Logging
	levels := getSupportedLevels()

	// set log level
	if !slices.Contains(levels, loggingLevel) {
		fmt.Printf("Incorrect logging level '%s', fallback to default value '%s'\n", loggingLevel, c.DefaultLogLevel)
		loggingLevel = c.DefaultLogLevel
	}

	// set log format
	loggingFormat := config.LogFormat
	if loggingFormat == "" {
		fmt.Printf("Unspecified log format, fallback to default log format '%s'\n", c.DefaultLogFormat)
		loggingFormat = c.DefaultLogFormat
	}

	flogging.Init(flogging.Config{
		Format:  string(loggingFormat),
		Writer:  os.Stderr,
		LogSpec: string(loggingLevel),
	})
}
