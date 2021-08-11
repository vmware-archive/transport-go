// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"github.com/vmware/transport-go/plank/utils"
)

// SpaConfig shorthand for SinglePageApplication Config is used to configure routes for your SPAs like
// Angular or React. for example if your app index.html is served at /app and static contents like JS/CSS
// are served from /app/static, BaseUri can be set to /app and StaticAssets to "/app/assets". see config.json
// for details.
type SpaConfig struct {
	RootFolder   string   `json:"root_folder"` // location where Plank will serve SPA
	BaseUri      string   `json:"base_uri"` // base URI for the SPA
	StaticAssets []string `json:"static_assets"` // locations for static assets used by the SPA
}

// NewSpaConfig takes location to where the SPA content is as an input and returns a sanitized
// instance of *SpaConfig.
func NewSpaConfig(input string) (spaConfig *SpaConfig, err error) {
	p, uri := utils.DeriveStaticURIFromPath(input)
	return &SpaConfig{
		RootFolder: p,
		BaseUri:    uri,
	}, nil
}
