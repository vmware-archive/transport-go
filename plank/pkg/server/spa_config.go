// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"github.com/vmware/transport-go/plank/utils"
)

type SpaConfig struct {
	RootFolder   string   `json:"root_folder"`
	BaseUri      string   `json:"base_uri"`
	StaticAssets []string `json:"static_assets"`
}

func NewSpaConfig(input string) (spaConfig *SpaConfig, err error) {
	p, uri := utils.DeriveStaticURIFromPath(input)
	return &SpaConfig{
		RootFolder: p,
		BaseUri:    uri,
	}, nil
}
