// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import "github.com/fatih/color"

var (
	InfoHeaderf  = color.New(color.FgHiBlue).Add(color.Bold).PrintfFunc()
	InfoHeaderFprintf = color.New(color.FgHiBlue).Add(color.Bold).FprintfFunc()
	Infof        = color.New(color.FgHiCyan).PrintfFunc()
	InfoFprintf        = color.New(color.FgHiCyan).FprintfFunc()
	WarnHeaderf  = color.New(color.FgHiYellow).Add(color.Bold).PrintfFunc()
	WarnHeaderFprintf  = color.New(color.FgHiYellow).Add(color.Bold).FprintfFunc()
	Warnf        = color.New(color.FgHiYellow).PrintfFunc()
	WarnFprintf        = color.New(color.FgHiYellow).FprintfFunc()
	ErrorHeaderf = color.New(color.FgHiRed).Add(color.Bold).PrintfFunc()
	ErrorHeaderFprintf = color.New(color.FgHiRed).Add(color.Bold).FprintfFunc()
	Errorf       = color.New(color.FgHiRed).PrintfFunc()
	ErrorFprintf       = color.New(color.FgHiRed).FprintfFunc()
)
