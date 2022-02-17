//go:build boot_img
// +build boot_img

package server

import "embed"

//go:embed logo.png
var logoFs embed.FS
