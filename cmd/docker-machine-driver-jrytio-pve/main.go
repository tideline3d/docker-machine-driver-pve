package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/rancher/machine/libmachine/drivers/plugin"
	"github.com/tideline3d/docker-machine-driver-jrytio-pve/cmd/docker-machine-driver-jrytio-pve/driver"
)

func main() {
	var (
		showHelp    = flag.Bool("help", false, "Show help and exit")
		showVersion = flag.Bool("version", false, "Show version information and exit")
	)

	flag.Parse()

	switch {
	case *showHelp:
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	case *showVersion:
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Fprint(os.Stderr, "Build info is not available")
			os.Exit(1)
		}

		fmt.Fprint(os.Stderr, buildInfo.Main.Version+"\n")
		os.Exit(0)
	default:
		plugin.RegisterDriver(driver.NewDriver("", ""))
	}
}
