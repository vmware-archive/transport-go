// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
)

var Log *PlankLogger

type LogConfig struct {
	AccessLog     string           `json:"access_log"`
	ErrorLog      string           `json:"error_log"`
	OutputLog     string           `json:"output_log"`
	FormatOptions *LogFormatOption `json:"format_options"`
	Root          string           `json:"root"`
	accessLogFp   io.Writer        `json:"-"`
	errorLogFp    io.Writer        `json:"-"`
	outputLogFp   io.Writer        `json:"-"`
}

// LogFormatOption is merely a wrapper of logrus.TextFormatter because TextFormatter does not allow serializing
// its public members of the struct
type LogFormatOption struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool `json:"force_colors"`

	// Force disabling colors.
	DisableColors bool `json:"disable_colors"`

	// Force quoting of all values
	ForceQuote bool `json:"force_quote"`

	// DisableQuote disables quoting for all values.
	// DisableQuote will have a lower priority than ForceQuote.
	// If both of them are set to true, quote will be forced on all values.
	DisableQuote bool `json:"disable_quote"`

	// Override coloring based on CLICOLOR and CLICOLOR_FORCE. - https://bixense.com/clicolors/
	EnvironmentOverrideColors bool `json:"environment_override_colors"`

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool `json:"disable_timestamp"`

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool `json:"full_timestamp"`

	// TimestampFormat to use for display when a full timestamp is printed
	TimestampFormat string `json:"timestamp_format"`

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool `json:"disable_sorting"`

	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool `json:"disable_level_truncation"`

	// PadLevelText Adds padding the level text so that all the levels output at the same length
	// PadLevelText is a superset of the DisableLevelTruncation option
	PadLevelText bool `json:"pad_level_text"`

	// QuoteEmptyFields will wrap empty fields in quotes if true
	QuoteEmptyFields bool `json:"quote_empty_fields"`

	// Whether the logger's out is to a terminal
	isTerminal bool `json:"is_terminal"`

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &TextFormatter{
	//     FieldMap: FieldMap{
	//         FieldKeyTime:  "@timestamp",
	//         FieldKeyLevel: "@level",
	//         FieldKeyMsg:   "@message"}}
	FieldMap logrus.FieldMap
}

func (lc *LogConfig) PrepareLogFiles() error {
	var fp io.Writer
	var err error
	if fp, err = lc.prepareLogFilePointer(lc.AccessLog); err != nil {
		return err
	}
	lc.accessLogFp = fp

	if fp, err = lc.prepareLogFilePointer(lc.ErrorLog); err != nil {
		return err
	}
	lc.errorLogFp = fp

	if fp, err = lc.prepareLogFilePointer(lc.OutputLog); err != nil {
		return err
	}
	lc.outputLogFp = fp

	return nil
}

func (lc *LogConfig) GetAccessLogFilePointer() io.Writer {
	return lc.accessLogFp
}

func (lc *LogConfig) GetErrorLogFilePointer() io.Writer {
	return lc.errorLogFp
}

func (lc *LogConfig) GetPlatformLogFilePointer() io.Writer {
	return lc.outputLogFp
}

func (lc *LogConfig) prepareLogFilePointer(target string) (fp io.Writer, err error) {
	if target == "stdout" {
		fp = os.Stdout
	} else if target == "stderr" {
		fp = os.Stderr
	} else if target == "null" {
		fp = &noopWriter{}
	} else {
		logFilePath := JoinBasePathIfRelativeRegularFilePath(lc.Root, target)
		fp, err = GetNewLogFilePointer(logFilePath)
	}
	return
}

type PlankLogger struct {
	*logrus.Logger
}

func (l *PlankLogger) setCommonFields() *logrus.Entry {
	fr := getFrame(2)
	pkgName := path.Base(path.Dir(fr.File))
	fileName := path.Base(fr.File)
	return l.WithFields(logrus.Fields{
		"goroutine": GetGoRoutineID(),
		"package":   pkgName,
		"fileName":  fileName,
	})
}

func (l *PlankLogger) Trace(args ...interface{}) {
	l.setCommonFields().Trace(args...)
}

func (l *PlankLogger) Traceln(args ...interface{}) {
	l.setCommonFields().Traceln(args...)
}

func (l *PlankLogger) Tracef(format string, args ...interface{}) {
	l.setCommonFields().Tracef(format, args...)
}

func (l *PlankLogger) Debug(args ...interface{}) {
	l.setCommonFields().Debug(args...)
}

func (l *PlankLogger) Debugln(args ...interface{}) {
	l.setCommonFields().Debugln(args...)
}

func (l *PlankLogger) Debugf(format string, args ...interface{}) {
	l.setCommonFields().Debugf(format, args...)
}

func (l *PlankLogger) Info(args ...interface{}) {
	l.setCommonFields().Info(args...)
}

func (l *PlankLogger) Infoln(args ...interface{}) {
	l.setCommonFields().Infoln(args...)
}

func (l *PlankLogger) Infof(format string, args ...interface{}) {
	l.setCommonFields().Infof(format, args...)
}

func (l *PlankLogger) Warn(args ...interface{}) {
	l.setCommonFields().Warn(args...)
}

func (l *PlankLogger) Warnln(args ...interface{}) {
	l.setCommonFields().Warnln(args...)
}

func (l *PlankLogger) Warnf(format string, args ...interface{}) {
	l.setCommonFields().Warnf(format, args...)
}

func (l *PlankLogger) Error(args ...interface{}) {
	l.setCommonFields().Error(args...)
}

func (l *PlankLogger) Errorln(args ...interface{}) {
	l.setCommonFields().Errorln(args...)
}

func (l *PlankLogger) Errorf(format string, args ...interface{}) {
	l.setCommonFields().Errorf(format, args...)
}

func (l *PlankLogger) Panic(args ...interface{}) {
	l.setCommonFields().Panic(args...)
}

func (l *PlankLogger) Panicln(args ...interface{}) {
	l.setCommonFields().Panicln(args...)
}

func (l *PlankLogger) Panicf(format string, args ...interface{}) {
	l.setCommonFields().Panicf(format, args...)
}

// noopWriter does absolutely nothing and return immediately. for references
// to those who wonder why this is here then in the first place, this is so
// this no-op writer instance can be passed as an access logger to not bother
// logging HTTP access logs which may not be necessary for some applications
// that want as lowest IO bottlenecks as possible.
type noopWriter struct{}

func (noopWriter *noopWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

func init() {
	Log = &PlankLogger{logrus.New()}
}

// CreateTextFormatterFromFormatOptions takes *server.LogFormatOption and returns
// the pointer to a new logrus.TextFormatter instance.
func CreateTextFormatterFromFormatOptions(opts *LogFormatOption) *logrus.TextFormatter {
	return &logrus.TextFormatter{
		ForceColors:               opts.ForceColors,
		DisableColors:             opts.DisableColors,
		ForceQuote:                opts.ForceQuote,
		DisableQuote:              opts.DisableQuote,
		EnvironmentOverrideColors: opts.EnvironmentOverrideColors,
		DisableTimestamp:          opts.DisableTimestamp,
		FullTimestamp:             opts.FullTimestamp,
		TimestampFormat:           opts.TimestampFormat,
		DisableSorting:            opts.DisableSorting,
		DisableLevelTruncation:    opts.DisableLevelTruncation,
		PadLevelText:              opts.PadLevelText,
		QuoteEmptyFields:          opts.QuoteEmptyFields,
		FieldMap:                  opts.FieldMap,
	}
}

// GetNewLogFilePointer returns the pointer to a new os.File instance given the file name
func GetNewLogFilePointer(file string) (*os.File, error) {
	return os.OpenFile(file, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
}
