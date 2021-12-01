package server

import (
	"embed"
	"fmt"
	imgcolor "image/color"
	"os"
	"runtime"

	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/fatih/color"
	"github.com/vmware/transport-go/plank/utils"
)

//go:embed logo.png
var logoFs embed.FS

// printBanner prints out banner as well as brief server config
func (ps *platformServer) printBanner() {
	// print out ascii art only if output is stdout
	if ps.out == os.Stdout {
		_, _ = fmt.Fprintf(ps.out, "\n\n")
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
	}

	// display title and config summary
	_, _ = fmt.Fprintln(ps.out)
	color.Set(color.BgHiWhite, color.FgHiBlack, color.Bold)
	_, _ = fmt.Fprintf(ps.out, " P L A N K ")
	color.Unset()
	_, _ = fmt.Fprintln(ps.out)
	utils.InfoFprintf(ps.out, "Host\t\t\t")
	_, _ = fmt.Fprintln(ps.out, ps.serverConfig.Host)
	utils.InfoFprintf(ps.out, "Port\t\t\t")
	_, _ = fmt.Fprintln(ps.out, ps.serverConfig.Port)

	if ps.serverConfig.FabricConfig != nil {
		utils.InfoFprintf(ps.out, "Fabric endpoint\t\t")
		if ps.serverConfig.FabricConfig.UseTCP {
			_, _ = fmt.Fprintln(ps.out, fmt.Sprintf(":%d (TCP)", ps.serverConfig.FabricConfig.TCPPort))
			return
		}
		_, _ = fmt.Fprintln(ps.out, ps.serverConfig.FabricConfig.FabricEndpoint)
	}

	if len(ps.serverConfig.StaticDir) > 0 {
		utils.InfoFprintf(ps.out, "Static endpoints\t")
		for i, dir := range ps.serverConfig.StaticDir {
			_, p := utils.DeriveStaticURIFromPath(dir)
			_, _ = fmt.Fprint(ps.out, p)
			if i < len(ps.serverConfig.StaticDir)-1 {
				_, _ = fmt.Fprint(ps.out, ", ")
			} else {
				_, _ = fmt.Fprint(ps.out, "\n")
			}
		}
	}

	if ps.serverConfig.SpaConfig != nil {
		utils.InfoFprintf(ps.out, "SPA endpoint\t\t")
		_, _ = fmt.Fprintln(ps.out, ps.serverConfig.SpaConfig.BaseUri)
		utils.InfoFprintf(ps.out, "SPA static assets\t")
		if len(ps.serverConfig.SpaConfig.StaticAssets) == 0 {
			_, _ = fmt.Fprint(ps.out, "-")
		}
		for idx, asset := range ps.serverConfig.SpaConfig.StaticAssets {
			_, uri := utils.DeriveStaticURIFromPath(asset)
			_, _ = fmt.Fprint(ps.out, utils.SanitizeUrl(uri, false))
			if idx < len(ps.serverConfig.SpaConfig.StaticAssets)-1 {
				_, _ = fmt.Fprint(ps.out, ", ")
			}
		}
		_, _ = fmt.Fprintln(ps.out)
	}

	utils.InfoFprintf(ps.out, "Health endpoint\t\t")
	_, _ = fmt.Fprintln(ps.out, "/health")

	if ps.serverConfig.EnablePrometheus {
		utils.InfoFprintf(ps.out, "Prometheus endpoint\t")
		_, _ = fmt.Fprintln(ps.out, "/prometheus")
	}

	_, _ = fmt.Fprintln(ps.out)

}
