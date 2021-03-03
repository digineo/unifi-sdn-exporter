package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"git.digineo.de/digineo/unifi-sdn-exporter/exporter"
	"git.digineo.de/digineo/unifi-sdn-exporter/unifi"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// DefaultConfigPath points to the default config file location.
// This might be overwritten at build time (using -ldflags).
var DefaultConfigPath = "./config.toml"

// nolint: gochecknoglobals
var (
	version = "development"
	commit  = "HEAD"
)

func main() {
	listenAddress := kingpin.Flag(
		"web.listen-address",
		"Address on which to expose metrics and web interface.",
	).Default(":9810").String()

	configFile := kingpin.Flag(
		"web.config",
		"Path to config.toml that contains all the targets.",
	).Default(DefaultConfigPath).String()

	verbose := kingpin.Flag(
		"verbose",
		"Increase verbosity",
	).Bool()

	kingpin.Flag("version", "Show version information").
		Short('v').
		PreAction(func(*kingpin.ParseContext) error {
			printVersion()
			os.Exit(0)
			return nil
		}).
		Bool()

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.SetFlags(log.Lshortfile)
	cfg, err := exporter.LoadConfig(*configFile)
	if err != nil {
		log.Fatal(err.Error())
	}

	unifi.Verbose = *verbose
	cfg.Start(*listenAddress, version)
}

func printVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	const l = "%-10s %-50s %s\n"
	v := fmt.Sprintf("%s (commit %s)\n", version, commit)
	fmt.Printf(l, "main", info.Main.Path, v)

	fmt.Println("Dependencies:")
	for _, i := range info.Deps {
		if r := i.Replace; r != nil {
			fmt.Printf(l, "dep", r.Path, r.Version)
			fmt.Printf(l, "  replaces", i.Path, i.Version)
		} else {
			fmt.Printf(l, "dep", i.Path, i.Version)
		}
	}
}
