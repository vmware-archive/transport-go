package log

import (
    "fmt"
    "github.com/fatih/color"
    "os"
    "strings"
)

// These flags have to be set from the "opts" module in each sewing-machine tool

var WarnFlag = true
var TraceFlag = true
var DebugFlag = true
var VerboseFlag = true
var RecoverOnError = true
var Version = ""

// Print warnings
func Warn(format string, arg ...interface{}) {
    color.NoColor = false
    color.Set(color.FgHiMagenta)
    if !WarnFlag {
        fmt.Printf("⚠️🚨 WARNING: "+format, arg...)
    }
    color.Unset()
}

// Print traces
func Trace(format string, arg ...interface{}) {
    color.NoColor = false
    color.Set(color.FgCyan)
    color.Set(color.Faint)
    if TraceFlag {
        fmt.Printf(format, arg...)
    }
    color.Unset()
}

// Print debug
func Debug(format string, arg ...interface{}) {
    if DebugFlag {
        fmt.Printf(format, arg...)
    }
}

// Print verbose
func Verbose(format string, arg ...interface{}) {
    color.NoColor = false
    color.Set(color.FgHiMagenta)
    if VerboseFlag {
        fmt.Printf(format, arg...)
    }
    color.Unset()
}

// Catchable Panic
func Panicf(format string, args ...interface{}) {
    color.NoColor = false
    color.Set(color.FgRed)
    color.Set(color.Bold)

    fmt.Printf("❌ FATAL: "+format, args...)
    color.Unset()
    if !RecoverOnError {
        os.Exit(4)
    }
}

func SetVersion(version string) {
    Version = version
    if strings.Contains(Version, "-") {
        Version = Version[:strings.Index(version, "-")]
    } else {
        Version = "v2.285"
    }
}
