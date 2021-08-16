// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import "github.com/fatih/color"

var (
	InfoHeaderf  = color.New(color.FgHiBlue).Add(color.Bold).PrintfFunc()
	Infof        = color.New(color.FgHiCyan).PrintfFunc()
	WarnHeaderf  = color.New(color.FgHiYellow).Add(color.Bold).PrintfFunc()
	Warnf        = color.New(color.FgHiYellow).PrintfFunc()
	ErrorHeaderf = color.New(color.FgHiRed).Add(color.Bold).PrintfFunc()
	Errorf       = color.New(color.FgHiRed).PrintfFunc()
)
