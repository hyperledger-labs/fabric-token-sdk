/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Ensure MockLogger satisfies the Logger interface
var _ Logger = (*MockLogger)(nil)

type MockLogger struct{}

func (m *MockLogger) Named(name string) Logger {
	fmt.Println("Named:", name)
	return m
}

func (m *MockLogger) Debug(args ...interface{}) {
	fmt.Println("DEBUG:", fmt.Sprint(args...))
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("DEBUG: "+format+"\n", args...)
}

func (m *MockLogger) Error(args ...interface{}) {
	fmt.Println("ERROR:", fmt.Sprint(args...))
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}

func (m *MockLogger) Fatal(args ...interface{}) {
	fmt.Println("FATAL:", fmt.Sprint(args...))
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	fmt.Printf("FATAL: "+format+"\n", args...)
}

func (m *MockLogger) Info(args ...interface{}) {
	fmt.Println("INFO:", fmt.Sprint(args...))
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("INFO: "+format+"\n", args...)
}

func (m *MockLogger) Panic(args ...interface{}) {
	fmt.Println("PANIC:", fmt.Sprint(args...))
}

func (m *MockLogger) Panicf(format string, args ...interface{}) {
	fmt.Printf("PANIC: "+format+"\n", args...)
}

func (m *MockLogger) Warn(args ...interface{}) {
	fmt.Println("WARN:", fmt.Sprint(args...))
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf("WARN: "+format+"\n", args...)
}

func (m *MockLogger) IsEnabledFor(level zapcore.Level) bool {
	// Implement logic to check if the given log level is enabled
	return true
}

func (m *MockLogger) Warnw(format string, args ...interface{}) {
	fmt.Printf("WARN: "+format+"\n", args...)
}

func (m *MockLogger) Warningf(format string, args ...interface{}) {
	fmt.Printf("WARNING: "+format+"\n", args...)
}

func (m *MockLogger) Errorw(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}

func (m *MockLogger) With(args ...interface{}) Logger {
	fmt.Println("With:", fmt.Sprint(args...))
	return m
}

func (m *MockLogger) Zap() *zap.Logger {
	return nil
}
