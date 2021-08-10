// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

var logMsg string
var logLevel logrus.Level

func TestPlankLogger(t *testing.T) {
	resetLogger()
	assert.NotNil(t, Log)
}

func TestPlankLogger_Debug(t *testing.T) {
	resetLogger()

	Log.Debug("debug")
	assert.EqualValues(t, "debug", logMsg)
	assert.EqualValues(t, logrus.DebugLevel, logLevel)
}

func TestPlankLogger_Info(t *testing.T) {
	resetLogger()

	Log.Info("info")
	assert.EqualValues(t, "info", logMsg)
	assert.EqualValues(t, logrus.InfoLevel, logLevel)
}

func TestPlankLogger_Warn(t *testing.T) {
	resetLogger()

	Log.Warn("warn")
	assert.EqualValues(t, "warn", logMsg)
	assert.EqualValues(t, logrus.WarnLevel, logLevel)
}

func TestPlankLogger_Error(t *testing.T) {
	resetLogger()

	Log.Error("error")
	assert.EqualValues(t, "error", logMsg)
	assert.EqualValues(t, logrus.ErrorLevel, logLevel)
}

func resetLogger() {
	logMsg = ""
	logLevel = logrus.TraceLevel

	Log = nil
	Log = &PlankLogger{logrus.New()}
	Log.SetLevel(logrus.DebugLevel)
	Log.AddHook(newTestHook(
		[]logrus.Level{logrus.DebugLevel, logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel},
		func(entry *logrus.Entry) error {
			logMsg = entry.Message
			logLevel = entry.Level
			return nil
		}))
}

func TestCreateTextFormatterFromFormatOptions(t *testing.T) {
	options := &LogFormatOption{
		ForceColors:               true,
		DisableColors:             true,
		ForceQuote:                true,
		DisableQuote:              true,
		EnvironmentOverrideColors: true,
		DisableTimestamp:          true,
		FullTimestamp:             true,
		TimestampFormat:           "Format",
		DisableSorting:            true,
		DisableLevelTruncation:    true,
		PadLevelText:              true,
		QuoteEmptyFields:          true,
		isTerminal:                true,
	}
	txtFormatter := CreateTextFormatterFromFormatOptions(options)
	assert.NotNil(t, txtFormatter)
}

// test models and utilities
type testHook struct {
	levels   []logrus.Level
	fireFunc func(entry *logrus.Entry) error
}

func (th *testHook) Levels() []logrus.Level {
	return th.levels
}

func (th *testHook) Fire(entry *logrus.Entry) error {
	return th.fireFunc(entry)
}

func (th *testHook) setFireFunc(f func(entry *logrus.Entry) error) {
	th.fireFunc = f
}

func newTestHook(levels []logrus.Level, firefunc func(entry *logrus.Entry) error) logrus.Hook {
	nh := &testHook{
		levels: levels,
	}
	nh.setFireFunc(firefunc)
	return nh
}
