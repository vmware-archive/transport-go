package server

import (
	"embed"
	"fmt"
	imgcolor "image/color"
	"runtime"

	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/fatih/color"
	"github.com/vmware/transport-go/plank/utils"
)

//go:embed logo.png
var logoFs embed.FS

// printBanner prints out banner as well as brief server config
func (ps *platformServer) printBanner() {
	fmt.Printf("\n\n")
	fs, _ := logoFs.Open("logo.png")
	defer fs.Close()
	img, err := ansimage.NewScaledFromReader(
		fs,
		40,
		80,
		imgcolor.Transparent,
		ansimage.ScaleModeResize,
		ansimage.NoDithering)
	if err != nil {
		panic(err)
	}
	// print out logo in ASCII art
	img.SetMaxProcs(runtime.NumCPU())
	img.Draw()

	// display title and config summary
	fmt.Println()
	color.Set(color.BgHiWhite, color.FgHiBlack, color.Bold)
	fmt.Printf(" P L A N K ")
	color.Unset()
	fmt.Println()
	utils.Infof("Host\t\t\t")
	fmt.Println(ps.serverConfig.Host)
	utils.Infof("Port\t\t\t")
	fmt.Println(ps.serverConfig.Port)

	if ps.serverConfig.FabricConfig != nil {
		utils.Infof("Fabric endpoint\t\t")
		fmt.Println(ps.serverConfig.FabricConfig.FabricEndpoint)
	}

	if len(ps.serverConfig.StaticDir) > 0 {
		utils.Infof("Static endpoints\t")
		for i, dir := range ps.serverConfig.StaticDir {
			_, p := utils.DeriveStaticURIFromPath(dir)
			fmt.Print(p)
			if i < len(ps.serverConfig.StaticDir)-1 {
				fmt.Print(", ")
			} else {
				fmt.Print("\n")
			}
		}
	}

	if ps.serverConfig.SpaConfig != nil {
		utils.Infof("SPA endpoint\t\t")
		fmt.Println(ps.serverConfig.SpaConfig.BaseUri)
		utils.Infof("SPA static assets\t")
		if len(ps.serverConfig.SpaConfig.StaticAssets) == 0 {
			fmt.Print("-")
		}
		for idx, asset := range ps.serverConfig.SpaConfig.StaticAssets {
			_, uri := utils.DeriveStaticURIFromPath(asset)
			fmt.Print(utils.SanitizeUrl(uri, false))
			if idx < len(ps.serverConfig.SpaConfig.StaticAssets)-1 {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}

	utils.Infof("Health endpoint\t\t")
	fmt.Println("/health")

	if ps.serverConfig.EnablePrometheus {
		utils.Infof("Prometheus endpoint\t")
		fmt.Println("/prometheus")
	}

	fmt.Println()

}
